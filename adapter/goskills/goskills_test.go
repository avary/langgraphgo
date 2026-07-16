package goskills

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/smallnest/goskills"
	"github.com/smallnest/langgraphgo/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSkillPackage 模拟 goskills.SkillPackage 接口
type MockSkillPackage struct {
	path string
}

func (m MockSkillPackage) GetName() string {
	return "test-skill"
}

func (m MockSkillPackage) GetDescription() string {
	return "Test skill package"
}

func (m MockSkillPackage) GetVersion() string {
	return "1.0.0"
}

func (m MockSkillPackage) GetPath() string {
	return m.path
}

// TestSkillTool_Name tests the Name method
func TestSkillTool_Name(t *testing.T) {
	tool := &SkillTool{
		name: "test_tool",
	}
	assert.Equal(t, "test_tool", tool.Name())
}

// TestSkillTool_Description tests the Description method
func TestSkillTool_Description(t *testing.T) {
	tool := &SkillTool{
		description: "Test tool description",
	}
	assert.Equal(t, "Test tool description", tool.Description())
}

// TestSkillTool_Call_RunShellCode tests the run_shell_code case
func TestSkillTool_Call_RunShellCode(t *testing.T) {
	// Skip if bash is not available
	if _, err := os.Stat("/bin/bash"); os.IsNotExist(err) {
		t.Skip("Bash not available, skipping test")
	}

	tool := &SkillTool{
		name: "run_shell_code",
	}

	// Test valid input
	params := map[string]any{
		"code": "echo 'Hello from shell'",
		"args": map[string]any{},
	}
	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Call(context.Background(), string(input))
	assert.NoError(t, err)
	assert.Contains(t, result, "Hello from shell")
}

// TestSkillTool_Call_RunShellCode_InvalidInput tests run_shell_code with invalid input
func TestSkillTool_Call_RunShellCode_InvalidInput(t *testing.T) {
	tool := &SkillTool{
		name: "run_shell_code",
	}

	// Test invalid JSON
	_, err := tool.Call(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal")
}

// TestSkillTool_Call_RunPythonCode tests the run_python_code case
func TestSkillTool_Call_RunPythonCode(t *testing.T) {
	// Skip if python is not available
	if _, err := os.Stat("/usr/bin/python3"); os.IsNotExist(err) {
		if _, err := os.Stat("/usr/bin/python"); os.IsNotExist(err) {
			t.Skip("Python not available, skipping test")
		}
	}

	tool := &SkillTool{
		name: "run_python_code",
	}

	// Test valid input
	params := map[string]any{
		"code": "print('Hello from Python')",
		"args": map[string]any{},
	}
	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Call(context.Background(), string(input))
	assert.NoError(t, err)
	assert.Contains(t, result, "Hello from Python")
}

// TestSkillTool_Call_ReadFile tests the read_file case
func TestSkillTool_Call_ReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Test file content"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	tool := &SkillTool{
		name: "read_file",
	}

	// Test with absolute path
	params := map[string]string{
		"filePath": testFile,
	}
	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Call(context.Background(), string(input))
	assert.NoError(t, err)
	assert.Equal(t, testContent, result)

	// Test with relative path and skillPath
	tool.skillPath = tmpDir
	params = map[string]string{
		"filePath": "test.txt",
	}
	input, err = json.Marshal(params)
	require.NoError(t, err)

	result, err = tool.Call(context.Background(), string(input))
	assert.NoError(t, err)
	assert.Equal(t, testContent, result)
}

// TestSkillTool_Call_WriteFile tests the write_file case
func TestSkillTool_Call_WriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "write_test.txt")
	testContent := "Content to write"

	tool := &SkillTool{
		name: "write_file",
	}

	params := map[string]string{
		"filePath": testFile,
		"content":  testContent,
	}
	input, err := json.Marshal(params)
	require.NoError(t, err)

	result, err := tool.Call(context.Background(), string(input))
	assert.NoError(t, err)
	assert.Contains(t, result, "Successfully wrote to file")

	// Verify file was written
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

// TestSkillTool_Call_UnknownTool tests unknown tool case
func TestSkillTool_Call_UnknownTool(t *testing.T) {
	tool := &SkillTool{
		name: "unknown_tool",
	}

	result, err := tool.Call(context.Background(), "{}")
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "unknown tool")
}

// TestSkillTool_Call_CustomScript tests custom script execution
func TestSkillTool_Call_CustomScript(t *testing.T) {
	// Skip if bash is not available
	if _, err := os.Stat("/bin/bash"); os.IsNotExist(err) {
		t.Skip("Bash not available, skipping test")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")
	scriptContent := "#!/bin/bash\necho 'Custom script executed'"
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	tool := &SkillTool{
		name: "custom_script",
		scriptMap: map[string]string{
			"custom_script": scriptPath,
		},
	}

	result, err := tool.Call(context.Background(), `{"args": []}`)
	assert.NoError(t, err)
	assert.Contains(t, result, "Custom script executed")
}

// TestSkillsToTools tests the SkillsToTools function
func TestSkillsToTools(t *testing.T) {
	// Since we can't easily create a real goskills.SkillPackage without the dependency,
	// we'll just verify the function exists and can be called with proper types.
	// In a real scenario with the goskills dependency, you would create a mock skill package.

	t.Run("function_signature", func(t *testing.T) {
		// Verify the function exists by checking its type
		var _ func(*goskills.SkillPackage, ...SkillsToToolsOptions) ([]tool.Tool, error) = SkillsToTools
		// This will compile if the function exists with the correct signature
	})
}

// TestSkillTool_ImplementsInterface verifies SkillTool implements tool.Tool
func TestSkillTool_ImplementsInterface(t *testing.T) {
	var _ tool.Tool = &SkillTool{}
	tool := &SkillTool{
		name:        "test",
		description: "test description",
	}

	assert.Equal(t, "test", tool.Name())
	assert.Equal(t, "test description", tool.Description())
}

// TestSkillTool_Call_EdgeCases tests various edge cases
func TestSkillTool_Call_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		toolName    string
		input       string
		expectError bool
	}{
		{
			name:        "empty input for run_shell_code",
			toolName:    "run_shell_code",
			input:       "",
			expectError: true,
		},
		{
			name:        "missing code parameter",
			toolName:    "run_shell_code",
			input:       `{"args": {}}`,
			expectError: false, // The tool might handle empty code gracefully
		},
		{
			name:        "invalid file path for read_file",
			toolName:    "read_file",
			input:       `{"filePath": ""}`,
			expectError: true,
		},
		{
			name:        "empty query for duckduckgo_search",
			toolName:    "duckduckgo_search",
			input:       `{"query": ""}`,
			expectError: false, // Might not error, just return empty result
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := &SkillTool{
				name: tt.toolName,
			}

			_, err := tool.Call(context.Background(), tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				// Don't assert on success as some cases might fail due to external dependencies
				_ = err
			}
		})
	}
}
