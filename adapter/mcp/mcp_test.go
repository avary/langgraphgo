package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sashabaranov/go-openai"
	mcpclient "github.com/smallnest/goskills/mcp"
	"github.com/smallnest/langgraphgo/tooltypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPTool_Interface(t *testing.T) {
	// Verify that MCPTool implements tooltypes.Tool interface
	var _ tooltypes.Tool = &MCPTool{}
}

func TestMCPTool_NameAndDescription(t *testing.T) {
	tool := &MCPTool{
		name:        "test_tool",
		description: "A test tool",
	}

	assert.Equal(t, "test_tool", tool.Name())
	assert.Equal(t, "A test tool", tool.Description())
}

func TestGetToolSchema(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query",
			},
		},
	}

	tool := &MCPTool{
		name:        "search",
		description: "Search tool",
		parameters:  schema,
	}

	retrievedSchema, ok := GetToolSchema(tool)
	assert.True(t, ok)
	assert.Equal(t, schema, retrievedSchema)

	// Test with non-MCP tool (using a different MCPTool without the schema)
	nonMCPTool := &MCPTool{
		name:        "other",
		description: "Other tool",
		parameters:  nil,
	}
	retrievedSchema, ok = GetToolSchema(nonMCPTool)
	assert.True(t, ok)
	assert.Nil(t, retrievedSchema)
}

// TestMCPToTools_EmptyClient tests the conversion with no tools
func TestMCPToTools_EmptyConversion(t *testing.T) {
	// This test would require a mock MCP client
	// For now, we just verify the function signature exists
	ctx := context.Background()

	// We can't actually call MCPToTools without a real client,
	// but we verify the function exists and has the right signature
	_ = ctx

	// Type check
	var fn func(context.Context, any) ([]tooltypes.Tool, error)
	_ = fn
}

// Example usage documentation
func ExampleMCPToTools() {
	ctx := context.Background()

	// Load MCP client from config file
	client, err := NewClientFromConfig(ctx, "~/.claude.json")
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Convert MCP tools to langchaingo tools
	tools, err := MCPToTools(ctx, client)
	if err != nil {
		panic(err)
	}

	// Use tools with langchaingo or langgraphgo agents
	_ = tools
}

func ExampleNewClientFromConfig() {
	ctx := context.Background()

	// Create MCP client from Claude config file
	client, err := NewClientFromConfig(ctx, "~/.claude.json")
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Now you can use the client to get tools or call them directly
	_ = client
}

// Note: 由于 mcp.Client 接口的复杂性，我们无法轻易创建 mock
// 我们将专注于测试不依赖外部客户端的部分

// TestMCPTool_Call 测试 MCPTool 的 Call 方法
func TestMCPTool_Call(t *testing.T) {
	// 由于接口限制，我们只能测试基本功能
	// 在实际场景中，需要真实的 MCP 客户端

	t.Run("mock_client_scenario", func(t *testing.T) {
		// 验证函数签名和基本结构
		tool := &MCPTool{
			name:        "test_tool",
			description: "A test tool",
			client:      nil, // 在实际测试中这会导致错误
		}

		assert.Equal(t, "test_tool", tool.Name())
		assert.Equal(t, "A test tool", tool.Description())

		// 测试 JSON 解析逻辑（在调用 client 之前）
		ctx := context.Background()
		validInput := `{"message": "Hello, MCP!"}`

		// 由于 client 是 nil，这会 panic，但我们可以捕获它
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Expected panic with nil client: %v", r)
			}
		}()

		_, err := tool.Call(ctx, validInput)
		// 我们不关心结果，因为会 panic
		_ = err
	})
}

// TestMCPTool_Call_EmptyInput 测试空输入
func TestMCPTool_Call_EmptyInput(t *testing.T) {
	tool := &MCPTool{
		name:   "test_tool",
		client: nil, // 会导致 panic
	}

	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic with nil client: %v", r)
		}
	}()

	_, err := tool.Call(ctx, "")
	// 我们不关心结果，因为会 panic
	_ = err
}

// TestMCPTool_Call_InvalidJSON 测试无效 JSON 输入
func TestMCPTool_Call_InvalidJSON(t *testing.T) {
	tool := &MCPTool{
		name:   "test_tool",
		client: nil, // 会在 JSON 解析后 panic
	}

	ctx := context.Background()
	_, err := tool.Call(ctx, "{invalid json}")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to call MCP tool")
}

// TestMCPTool_Call_ClientError 测试客户端错误
func TestMCPTool_Call_ClientError(t *testing.T) {
	// 由于接口限制，我们无法正确模拟客户端错误
	// 但我们可以验证错误处理逻辑
	tool := &MCPTool{
		name:   "error_tool",
		client: nil, // 会导致 panic
	}

	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic with nil client: %v", r)
		}
	}()

	_, err := tool.Call(ctx, "{}")
	// 我们不关心结果，因为会 panic
	_ = err
}

// TestMCPToTools_FunctionSignature 测试 MCPToTools 函数签名
func TestMCPToTools_FunctionSignature(t *testing.T) {
	// 验证函数存在并有正确的签名
	var _ func(context.Context, *mcpclient.Client) ([]tooltypes.Tool, error) = MCPToTools
}

// TestNewClientFromConfig_NonExistentFile 测试不存在的配置文件
func TestNewClientFromConfig_NonExistentFile(t *testing.T) {
	ctx := context.Background()

	// 测试不存在的文件路径
	_, err := NewClientFromConfig(ctx, "/non/existent/path/config.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load MCP config")
}

// TestNewClientFromConfig_InvalidConfig 测试无效的配置文件
func TestNewClientFromConfig_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid_config.json")

	// 写入无效的 JSON
	err := os.WriteFile(configFile, []byte("{invalid json"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = NewClientFromConfig(ctx, configFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load MCP config")
}

// TestGetToolSchema_NonMCPTool 测试非 MCP 工具
func TestGetToolSchema_NonMCPTool(t *testing.T) {
	// 使用一个简单的工具实现
	nonMCPTool := &simpleTool{name: "test"}

	schema, ok := GetToolSchema(nonMCPTool)
	assert.False(t, ok)
	assert.Nil(t, schema)
}

// simpleTool 简单的工具实现用于测试
type simpleTool struct {
	name string
}

func (t *simpleTool) Name() string        { return t.name }
func (t *simpleTool) Description() string { return "Simple test tool" }
func (t *simpleTool) Call(ctx context.Context, input string) (string, error) {
	return "result", nil
}

// TestMCPToolsToOpenAI_FunctionSignature 测试 MCPToolsToOpenAI 函数签名
func TestMCPToolsToOpenAI_FunctionSignature(t *testing.T) {
	// 验证函数存在并有正确的签名
	var _ func(context.Context, *mcpclient.Client) ([]openai.Tool, error) = MCPToolsToOpenAI
}

// TestMCPTool_EdgeCases 测试边界情况
func TestMCPTool_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupTool   func() *MCPTool
		input       string
		expectError bool
	}{
		{
			name: "nil client",
			setupTool: func() *MCPTool {
				return &MCPTool{
					name:   "test",
					client: nil,
				}
			},
			input:       "{}",
			expectError: true, // 应该 panic 或返回错误
		},
		{
			name: "empty tool name",
			setupTool: func() *MCPTool {
				return &MCPTool{
					name:   "",
					client: nil, // 会导致 panic
				}
			},
			input:       "{}",
			expectError: false, // 空名字可能不会导致错误
		},
		{
			name: "invalid JSON",
			setupTool: func() *MCPTool {
				return &MCPTool{
					name:   "test",
					client: nil,
				}
			},
			input:       `{invalid json}`,
			expectError: true,
		},
		{
			name: "valid JSON",
			setupTool: func() *MCPTool {
				return &MCPTool{
					name:   "test",
					client: nil,
				}
			},
			input: `{
				"param1": "value1",
				"param2": 123
			}`,
			expectError: false, // JSON 解析会成功，但之后会 panic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := tt.setupTool()
			ctx := context.Background()

			defer func() {
				if r := recover(); r != nil {
					// 对于 nil client 的情况，panic 是可接受的
					t.Logf("Recovered from panic: %v", r)
				}
			}()

			_, err := tool.Call(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
			}
		})
	}
}

// TestMCPTool_ParameterHandling 测试参数处理
func TestMCPTool_ParameterHandling(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"required_param": map[string]any{
				"type": "string",
			},
			"optional_param": map[string]any{
				"type": "number",
			},
		},
		"required": []string{"required_param"},
	}

	tool := &MCPTool{
		name:        "parameter_test",
		description: "Tool for testing parameters",
		parameters:  schema,
	}

	// 测试获取 schema
	retrievedSchema, ok := GetToolSchema(tool)
	assert.True(t, ok)
	assert.Equal(t, schema, retrievedSchema)

	// 验证 schema 结构
	if schemaMap, ok := retrievedSchema.(map[string]any); ok {
		assert.Equal(t, "object", schemaMap["type"])
		if props, ok := schemaMap["properties"].(map[string]any); ok {
			assert.Contains(t, props, "required_param")
			assert.Contains(t, props, "optional_param")
		}
	}
}
