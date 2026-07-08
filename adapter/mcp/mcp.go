package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
	mcpclient "github.com/smallnest/goskills/mcp"
	"github.com/smallnest/langgraphgo/tooltypes"
)

// MCPTool implements tooltypes.Tool for MCP (Model Context Protocol) tools.
type MCPTool struct {
	name        string
	description string
	client      *mcpclient.Client
	parameters  any // JSON schema for the tool parameters
}

var _ tooltypes.Tool = &MCPTool{}

func (t *MCPTool) Name() string {
	return t.name
}

func (t *MCPTool) Description() string {
	return t.description
}

func (t *MCPTool) Call(ctx context.Context, input string) (string, error) {
	// Parse input JSON into a map
	var args map[string]any
	if input != "" {
		if err := json.Unmarshal([]byte(input), &args); err != nil {
			// 如果 JSON 解析失败，尝试作为纯文本参数传递
			// 有些 MCP 工具可能接受简单的字符串参数
			args = map[string]any{"input": input}
		}
	} else {
		// 空输入时使用空对象
		args = make(map[string]any)
	}

	// Call the MCP tool through the client
	result, err := t.client.CallTool(ctx, t.name, args)
	if err != nil {
		return "", fmt.Errorf("failed to call MCP tool %s: %w", t.name, err)
	}

	// Convert result to JSON string
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP tool result: %w", err)
	}

	return string(resultJSON), nil
}

// MCPToTools converts MCP tools from a client to langchaingo tools.
// It fetches all available tools from the connected MCP servers and wraps them
// as langchaingo tooltypes.Tool instances.
func MCPToTools(ctx context.Context, client *mcpclient.Client) ([]tooltypes.Tool, error) {
	// Get all OpenAI tools from MCP servers
	openaiTools, err := client.GetTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP tools: %w", err)
	}

	// Convert to langchaingo tools
	var result []tooltypes.Tool
	for _, t := range openaiTools {
		if t.Function == nil || t.Function.Name == "" {
			continue
		}

		result = append(result, &MCPTool{
			name:        t.Function.Name,
			description: t.Function.Description,
			client:      client,
			parameters:  t.Function.Parameters,
		})
	}

	return result, nil
}

// NewClientFromConfig creates a new MCP client from a config file path.
// This is a convenience function that loads the config and creates a client.
func NewClientFromConfig(ctx context.Context, configPath string) (*mcpclient.Client, error) {
	config, err := mcpclient.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load MCP config: %w", err)
	}

	client, err := mcpclient.NewClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	return client, nil
}

// GetToolSchema returns the JSON schema for a tool's parameters.
// This can be useful for debugging or generating documentation.
func GetToolSchema(tool tooltypes.Tool) (any, bool) {
	if mcpTool, ok := tool.(*MCPTool); ok {
		return mcpTool.parameters, true
	}
	return nil, false
}

// MCPToolsToOpenAI converts MCP tools to OpenAI tool definitions.
// This is useful when you need to use MCP tools directly with OpenAI's API.
func MCPToolsToOpenAI(ctx context.Context, client *mcpclient.Client) ([]openai.Tool, error) {
	return client.GetTools(ctx)
}
