package openairesponses

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/smallnest/langgraphgo/llms"
)

// Call generates a completion from a single prompt.
func (o *LLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, o, prompt, options...)
}

// GenerateContent generates a completion from a sequence of messages via the
// OpenAI Responses API.
func (o *LLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	opts := llms.CallOptions{Model: o.model}
	for _, apply := range options {
		apply(&opts)
	}

	streaming := opts.StreamingFunc != nil
	req := o.buildRequest(messages, &opts, streaming)

	httpResp, err := o.post(ctx, req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, o.decodeError(httpResp)
	}

	if streaming {
		return o.parseStream(ctx, httpResp.Body, opts.StreamingFunc)
	}

	var resp response
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("openairesponses: decode response: %w", err)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	if len(resp.Output) == 0 {
		return nil, llms.ErrEmptyResponse
	}
	return resp.toContentResponse(), nil
}

func (o *LLM) post(ctx context.Context, body request) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openairesponses: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/responses", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.token)
	if o.orgID != "" {
		httpReq.Header.Set("OpenAI-Organization", o.orgID)
	}
	if body.Stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}

	return o.httpClient.Do(httpReq)
}

func (o *LLM) decodeError(resp *http.Response) error {
	var envelope struct {
		Error *responseError `json:"error"`
	}
	data, _ := io.ReadAll(resp.Body)
	if json.Unmarshal(data, &envelope) == nil && envelope.Error != nil {
		return envelope.Error
	}
	return fmt.Errorf("openairesponses: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
}

// parseStream consumes the SSE event stream, forwarding text deltas to
// streamingFunc and returning the final response assembled from the completed
// event (falling back to accumulated text if none arrives).
func (o *LLM) parseStream(ctx context.Context, body io.Reader, streamingFunc func(context.Context, []byte) error) (*llms.ContentResponse, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var accumulated strings.Builder
	var final *response

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		var evt streamEvent
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			continue
		}
		switch evt.Type {
		case "response.output_text.delta":
			accumulated.WriteString(evt.Delta)
			if err := streamingFunc(ctx, []byte(evt.Delta)); err != nil {
				return nil, err
			}
		case "response.completed":
			final = evt.Response
		case "error", "response.failed":
			if evt.Response != nil && evt.Response.Error != nil {
				return nil, evt.Response.Error
			}
			return nil, fmt.Errorf("openairesponses: stream error")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("openairesponses: read stream: %w", err)
	}

	if final != nil {
		return final.toContentResponse(), nil
	}
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{{Content: accumulated.String(), StopReason: "completed"}},
	}, nil
}

type streamEvent struct {
	Type     string    `json:"type"`
	Delta    string    `json:"delta"`
	Response *response `json:"response,omitempty"`
}
