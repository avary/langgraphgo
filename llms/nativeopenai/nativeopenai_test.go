package nativeopenai

import (
	"testing"

	goopenai "github.com/sashabaranov/go-openai"
	"github.com/smallnest/langgraphgo/llmtypes"
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
	cases := map[llmtypes.ChatMessageType]string{
		llmtypes.ChatMessageTypeSystem:   goopenai.ChatMessageRoleSystem,
		llmtypes.ChatMessageTypeAI:       goopenai.ChatMessageRoleAssistant,
		llmtypes.ChatMessageTypeTool:     goopenai.ChatMessageRoleTool,
		llmtypes.ChatMessageTypeFunction: goopenai.ChatMessageRoleFunction,
		llmtypes.ChatMessageTypeHuman:    goopenai.ChatMessageRoleUser,
		llmtypes.ChatMessageTypeGeneric:  goopenai.ChatMessageRoleUser,
	}
	for in, want := range cases {
		if got := toOpenAIRole(in); got != want {
			t.Errorf("role %q: want %q, got %q", in, want, got)
		}
	}
}

func TestToOpenAIMessages(t *testing.T) {
	msgs := []llmtypes.MessageContent{
		{Role: llmtypes.ChatMessageTypeSystem, Parts: []llmtypes.ContentPart{llmtypes.TextContent{Text: "be brief"}}},
		{Role: llmtypes.ChatMessageTypeHuman, Parts: []llmtypes.ContentPart{
			llmtypes.TextContent{Text: "describe this"},
			llmtypes.ImageURLContent{URL: "http://img", Detail: "low"},
		}},
		{Role: llmtypes.ChatMessageTypeAI, Parts: []llmtypes.ContentPart{
			llmtypes.ToolCall{ID: "c1", Type: "function", FunctionCall: &llmtypes.FunctionCall{Name: "f", Arguments: "{}"}},
		}},
		{Role: llmtypes.ChatMessageTypeTool, Parts: []llmtypes.ContentPart{
			llmtypes.ToolCallResponse{ToolCallID: "c1", Name: "f", Content: "42"},
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
	got := toOpenAIMessages([]llmtypes.MessageContent{
		{Role: llmtypes.ChatMessageTypeAI, Parts: []llmtypes.ContentPart{
			llmtypes.TextContent{Text: "let me search"},
			llmtypes.ToolCall{ID: "c1", Type: "function", FunctionCall: &llmtypes.FunctionCall{Name: "f", Arguments: "{}"}},
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
	tools := []llmtypes.Tool{{
		Type: "function",
		Function: &llmtypes.FunctionDefinition{
			Name:        "search",
			Description: "search the web",
			Parameters:  map[string]any{"type": "object"},
		},
	}}
	out := toOpenAITools(tools)
	if len(out) != 1 || out[0].Function == nil || out[0].Function.Name != "search" {
		t.Fatalf("tool not converted: %+v", out)
	}

	choice := toOpenAIToolChoice(llmtypes.ToolChoice{Type: "function", Function: &llmtypes.FunctionReference{Name: "search"}})
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
