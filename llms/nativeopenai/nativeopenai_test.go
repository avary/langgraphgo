package nativeopenai

import (
	"testing"

	goopenai "github.com/sashabaranov/go-openai"
	"github.com/smallnest/langgraphgo/llms"
)

func TestNew(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "")
		if _, err := New(); err != ErrMissingToken {
			t.Fatalf("want ErrMissingToken, got %v", err)
		}
	})

	t.Run("with token uses default model", func(t *testing.T) {
		llm, err := New(WithToken("sk-test"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if llm.model != DefaultModel {
			t.Fatalf("want default model %q, got %q", DefaultModel, llm.model)
		}
	})

	t.Run("options override", func(t *testing.T) {
		llm, err := New(WithToken("sk-test"), WithModel("gpt-4o"), WithBaseURL("https://example.com/v1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if llm.model != "gpt-4o" {
			t.Fatalf("want model gpt-4o, got %q", llm.model)
		}
	})
}

func TestToOpenAIRole(t *testing.T) {
	cases := map[llms.ChatMessageType]string{
		llms.ChatMessageTypeSystem:   goopenai.ChatMessageRoleSystem,
		llms.ChatMessageTypeAI:       goopenai.ChatMessageRoleAssistant,
		llms.ChatMessageTypeTool:     goopenai.ChatMessageRoleTool,
		llms.ChatMessageTypeFunction: goopenai.ChatMessageRoleFunction,
		llms.ChatMessageTypeHuman:    goopenai.ChatMessageRoleUser,
		llms.ChatMessageTypeGeneric:  goopenai.ChatMessageRoleUser,
	}
	for in, want := range cases {
		if got := toOpenAIRole(in); got != want {
			t.Errorf("role %q: want %q, got %q", in, want, got)
		}
	}
}

func TestToOpenAIMessages(t *testing.T) {
	msgs := []llms.MessageContent{
		{Role: llms.ChatMessageTypeSystem, Parts: []llms.ContentPart{llms.TextContent{Text: "be brief"}}},
		{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{
			llms.TextContent{Text: "describe this"},
			llms.ImageURLContent{URL: "http://img", Detail: "low"},
		}},
		{Role: llms.ChatMessageTypeAI, Parts: []llms.ContentPart{
			llms.ToolCall{ID: "c1", Type: "function", FunctionCall: &llms.FunctionCall{Name: "f", Arguments: "{}"}},
		}},
		{Role: llms.ChatMessageTypeTool, Parts: []llms.ContentPart{
			llms.ToolCallResponse{ToolCallID: "c1", Name: "f", Content: "42"},
		}},
	}

	got := toOpenAIMessages(msgs)
	if len(got) != 4 {
		t.Fatalf("want 4 messages, got %d", len(got))
	}

	if got[0].Content != "be brief" {
		t.Errorf("single text part should set Content, got %q", got[0].Content)
	}
	if len(got[1].MultiContent) != 2 {
		t.Errorf("mixed parts should produce MultiContent, got %d parts", len(got[1].MultiContent))
	}
	if len(got[2].ToolCalls) != 1 || got[2].ToolCalls[0].Function.Name != "f" {
		t.Errorf("assistant tool call not converted: %+v", got[2].ToolCalls)
	}
	if got[3].ToolCallID != "c1" || got[3].Content != "42" {
		t.Errorf("tool response not converted: %+v", got[3])
	}
}

func TestToOpenAIMessagesTextWithToolCall(t *testing.T) {
	// An assistant turn with both text and a tool call must keep string
	// Content: the Chat Completions API rejects array content for assistant.
	got := toOpenAIMessages([]llms.MessageContent{
		{Role: llms.ChatMessageTypeAI, Parts: []llms.ContentPart{
			llms.TextContent{Text: "let me search"},
			llms.ToolCall{ID: "c1", Type: "function", FunctionCall: &llms.FunctionCall{Name: "f", Arguments: "{}"}},
		}},
	})
	if len(got) != 1 {
		t.Fatalf("want 1 message, got %d", len(got))
	}
	if got[0].Content != "let me search" {
		t.Errorf("want string Content, got %q", got[0].Content)
	}
	if len(got[0].MultiContent) != 0 {
		t.Errorf("want no MultiContent, got %d parts", len(got[0].MultiContent))
	}
	if len(got[0].ToolCalls) != 1 {
		t.Errorf("want tool call preserved, got %+v", got[0].ToolCalls)
	}
}

func TestToolConversion(t *testing.T) {
	tools := []llms.Tool{{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "search",
			Description: "search the web",
			Parameters:  map[string]any{"type": "object"},
		},
	}}
	out := toOpenAITools(tools)
	if len(out) != 1 || out[0].Function == nil || out[0].Function.Name != "search" {
		t.Fatalf("tool not converted: %+v", out)
	}

	choice := toOpenAIToolChoice(llms.ToolChoice{Type: "function", Function: &llms.FunctionReference{Name: "search"}})
	tc, ok := choice.(goopenai.ToolChoice)
	if !ok || tc.Function.Name != "search" {
		t.Fatalf("tool choice not converted: %+v", choice)
	}

	if passthrough := toOpenAIToolChoice("auto"); passthrough != "auto" {
		t.Errorf("string tool choice should pass through, got %v", passthrough)
	}
}

func TestFromOpenAIToolCalls(t *testing.T) {
	if fromOpenAIToolCalls(nil) != nil {
		t.Error("nil input should return nil")
	}
	got := fromOpenAIToolCalls([]goopenai.ToolCall{{
		ID:       "c1",
		Type:     goopenai.ToolTypeFunction,
		Function: goopenai.FunctionCall{Name: "f", Arguments: `{"x":1}`},
	}})
	if len(got) != 1 || got[0].FunctionCall.Name != "f" || got[0].FunctionCall.Arguments != `{"x":1}` {
		t.Fatalf("tool call not converted: %+v", got)
	}
}
