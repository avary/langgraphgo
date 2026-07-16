package openairesponses

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/smallnest/langgraphgo/llms"
)

func TestNew(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "")
		if _, err := New(); err != ErrMissingToken {
			t.Fatalf("want ErrMissingToken, got %v", err)
		}
	})

	t.Run("defaults", func(t *testing.T) {
		llm, err := New(WithToken("sk-test"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if llm.model != DefaultModel {
			t.Fatalf("want default model %q, got %q", DefaultModel, llm.model)
		}
		if llm.baseURL != DefaultBaseURL {
			t.Fatalf("want default base %q, got %q", DefaultBaseURL, llm.baseURL)
		}
	})

	t.Run("options override and trim base url", func(t *testing.T) {
		llm, err := New(WithToken("sk-test"), WithModel("gpt-5"), WithBaseURL("https://example.com/v1/"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if llm.model != "gpt-5" {
			t.Fatalf("want model gpt-5, got %q", llm.model)
		}
		if llm.baseURL != "https://example.com/v1" {
			t.Fatalf("want trimmed base url, got %q", llm.baseURL)
		}
	})
}

func TestToInputItems(t *testing.T) {
	msgs := []llms.MessageContent{
		{Role: llms.ChatMessageTypeSystem, Parts: []llms.ContentPart{llms.TextContent{Text: "be brief"}}},
		{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{
			llms.TextContent{Text: "hi"},
			llms.ImageURLContent{URL: "http://img"},
		}},
		{Role: llms.ChatMessageTypeAI, Parts: []llms.ContentPart{
			llms.ToolCall{ID: "call_1", Type: "function", FunctionCall: &llms.FunctionCall{Name: "f", Arguments: "{}"}},
		}},
		{Role: llms.ChatMessageTypeTool, Parts: []llms.ContentPart{
			llms.ToolCallResponse{ToolCallID: "call_1", Content: "result"},
		}},
	}
	items := toInputItems(msgs)
	if len(items) != 4 {
		t.Fatalf("want 4 items, got %d", len(items))
	}
	if items[0].Role != "system" || items[0].Content[0].Type != "input_text" {
		t.Errorf("system item wrong: %+v", items[0])
	}
	if items[1].Content[1].Type != "input_image" || items[1].Content[1].ImageURL != "http://img" {
		t.Errorf("image part wrong: %+v", items[1].Content[1])
	}
	if items[2].Type != "function_call" || items[2].CallID != "call_1" || items[2].Name != "f" {
		t.Errorf("function_call item wrong: %+v", items[2])
	}
	if items[3].Type != "function_call_output" || items[3].Output != "result" {
		t.Errorf("function_call_output item wrong: %+v", items[3])
	}
}

func TestToInputItemsOrdering(t *testing.T) {
	// An assistant turn carrying both text and a tool call must flatten as
	// [message, function_call] so the text precedes the call it introduces.
	items := toInputItems([]llms.MessageContent{
		{Role: llms.ChatMessageTypeAI, Parts: []llms.ContentPart{
			llms.TextContent{Text: "let me look that up"},
			llms.ToolCall{ID: "call_1", Type: "function", FunctionCall: &llms.FunctionCall{Name: "f", Arguments: "{}"}},
		}},
	})
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	if items[0].Type != "message" || items[1].Type != "function_call" {
		t.Errorf("want [message, function_call], got [%s, %s]", items[0].Type, items[1].Type)
	}
}

func TestGenerateContent(t *testing.T) {
	var captured request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Errorf("want path /responses, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("want auth header, got %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{
			"id": "resp_1",
			"status": "completed",
			"output": [
				{"type": "message", "role": "assistant", "content": [{"type": "output_text", "text": "hello there"}]},
				{"type": "function_call", "call_id": "call_9", "name": "lookup", "arguments": "{\"q\":\"x\"}"}
			],
			"usage": {"input_tokens": 3, "output_tokens": 5, "total_tokens": 8}
		}`)
	}))
	defer srv.Close()

	llm, err := New(WithToken("sk-test"), WithModel("gpt-5"), WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp, err := llm.GenerateContent(context.Background(), []llms.MessageContent{
		{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextContent{Text: "hi"}}},
	}, llms.WithTools([]llms.Tool{
		{Type: "function", Function: &llms.FunctionDefinition{Name: "lookup", Description: "d"}},
	}))
	if err != nil {
		t.Fatalf("GenerateContent: %v", err)
	}

	if captured.Model != "gpt-5" {
		t.Errorf("want model gpt-5, got %q", captured.Model)
	}
	if len(captured.Input) != 1 || captured.Input[0].Type != "message" {
		t.Errorf("unexpected input: %+v", captured.Input)
	}
	if len(captured.Tools) != 1 || captured.Tools[0].Name != "lookup" {
		t.Errorf("unexpected tools: %+v", captured.Tools)
	}

	choice := resp.Choices[0]
	if choice.Content != "hello there" {
		t.Errorf("want content, got %q", choice.Content)
	}
	if len(choice.ToolCalls) != 1 || choice.ToolCalls[0].ID != "call_9" {
		t.Errorf("unexpected tool calls: %+v", choice.ToolCalls)
	}
	if choice.FuncCall == nil || choice.FuncCall.Name != "lookup" {
		t.Errorf("want FuncCall lookup, got %+v", choice.FuncCall)
	}
	if choice.GenerationInfo["total_tokens"] != 8 {
		t.Errorf("want total_tokens 8, got %v", choice.GenerationInfo["total_tokens"])
	}
}

func TestGenerateContentStreaming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hel\"}\n\n")
		io.WriteString(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"lo\"}\n\n")
		io.WriteString(w, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"r\",\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"hello\"}]}]}}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	llm, err := New(WithToken("sk-test"), WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks strings.Builder
	resp, err := llm.GenerateContent(context.Background(), []llms.MessageContent{
		{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextContent{Text: "hi"}}},
	}, llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
		chunks.Write(chunk)
		return nil
	}))
	if err != nil {
		t.Fatalf("GenerateContent streaming: %v", err)
	}
	if chunks.String() != "hello" {
		t.Errorf("want streamed chunks hello, got %q", chunks.String())
	}
	if resp.Choices[0].Content != "hello" {
		t.Errorf("want final content hello, got %q", resp.Choices[0].Content)
	}
}

func TestGenerateContentError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": {"message": "bad model", "type": "invalid_request_error"}}`)
	}))
	defer srv.Close()

	llm, err := New(WithToken("sk-test"), WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = llm.GenerateContent(context.Background(), []llms.MessageContent{
		{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextContent{Text: "hi"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "bad model") {
		t.Fatalf("want bad model error, got %v", err)
	}
}
