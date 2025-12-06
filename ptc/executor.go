package ptc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmc/langchaingo/tools"
)

// ExecutionLanguage defines the programming language for code execution
type ExecutionLanguage string

const (
	LanguagePython ExecutionLanguage = "python"
	LanguageGo     ExecutionLanguage = "go"
)

// ExecutionMode defines how tools are executed in the code
type ExecutionMode string

const (
	// ModeServer: Tools are called via HTTP server (default, recommended)
	// - Fully implemented and tested
	// - Better isolation (sandboxed)
	// - Reliable tool execution
	ModeServer ExecutionMode = "server"

	// ModeDirect: Tools are directly embedded in code (experimental)
	// - Currently uses placeholder implementations
	// - Not recommended for production use
	// - Work in progress
	ModeDirect ExecutionMode = "direct"
)

// CodeExecutor handles the execution of programmatic tool calling code
type CodeExecutor struct {
	Language   ExecutionLanguage
	Tools      []tools.Tool
	Timeout    time.Duration
	WorkDir    string
	Mode       ExecutionMode
	toolServer *ToolServer
}

// ExecutionResult contains the result of code execution
type ExecutionResult struct {
	Output string
	Error  error
	Stdout string
	Stderr string
}

// NewCodeExecutor creates a new code executor for PTC
// Default mode is ModeServer for reliability
func NewCodeExecutor(language ExecutionLanguage, toolList []tools.Tool) *CodeExecutor {
	return NewCodeExecutorWithMode(language, toolList, ModeServer)
}

// NewCodeExecutorWithMode creates a new code executor with specified execution mode
func NewCodeExecutorWithMode(language ExecutionLanguage, toolList []tools.Tool, mode ExecutionMode) *CodeExecutor {
	executor := &CodeExecutor{
		Language: language,
		Tools:    toolList,
		Timeout:  5 * time.Minute,
		WorkDir:  os.TempDir(),
		Mode:     mode,
	}

	// Only create tool server in server mode
	if mode == ModeServer {
		executor.toolServer = NewToolServer(toolList)
	}

	return executor
}

// Start starts the code executor and its tool server (only in server mode)
func (ce *CodeExecutor) Start(ctx context.Context) error {
	if ce.Mode == ModeServer && ce.toolServer != nil {
		return ce.toolServer.Start(ctx)
	}
	return nil
}

// Stop stops the code executor and its tool server (only in server mode)
func (ce *CodeExecutor) Stop(ctx context.Context) error {
	if ce.Mode == ModeServer && ce.toolServer != nil {
		return ce.toolServer.Stop(ctx)
	}
	return nil
}

// GetToolServerURL returns the URL of the tool server (empty in direct mode)
func (ce *CodeExecutor) GetToolServerURL() string {
	if ce.Mode == ModeServer && ce.toolServer != nil {
		return ce.toolServer.GetBaseURL()
	}
	return ""
}

// Execute runs the generated code with access to tools
func (ce *CodeExecutor) Execute(ctx context.Context, code string) (*ExecutionResult, error) {
	switch ce.Language {
	case LanguagePython:
		return ce.executePython(ctx, code)
	case LanguageGo:
		return ce.executeGo(ctx, code)
	default:
		return nil, fmt.Errorf("unsupported language: %s", ce.Language)
	}
}

// executePython executes Python code with tool bindings
func (ce *CodeExecutor) executePython(ctx context.Context, code string) (*ExecutionResult, error) {
	// Create a temporary Python script
	scriptPath := filepath.Join(ce.WorkDir, fmt.Sprintf("ptc_script_%d.py", time.Now().UnixNano()))
	defer os.Remove(scriptPath)

	// Generate Python tool wrapper functions based on execution mode
	var toolWrappers string
	if ce.Mode == ModeServer {
		toolWrappers = ce.generatePythonToolWrappersServer()
	} else {
		toolWrappers = ce.generatePythonToolWrappersDirect()
	}

	// Combine tool wrappers and user code
	fullScript := fmt.Sprintf(`
import json
import sys

# Tool wrapper functions
%s

# User code
%s
`, toolWrappers, code)

	if err := os.WriteFile(scriptPath, []byte(fullScript), 0644); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	// Execute Python script
	execCtx, cancel := context.WithTimeout(ctx, ce.Timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "python3", scriptPath)
	output, err := cmd.CombinedOutput()

	result := &ExecutionResult{
		Output: string(output),
		Stdout: string(output),
	}

	if err != nil {
		result.Error = err
	}

	return result, nil
}

// executeGo executes Go code with tool bindings
func (ce *CodeExecutor) executeGo(ctx context.Context, code string) (*ExecutionResult, error) {
	// Create a temporary Go file
	scriptPath := filepath.Join(ce.WorkDir, fmt.Sprintf("ptc_script_%d.go", time.Now().UnixNano()))
	defer os.Remove(scriptPath)

	// Generate Go tool wrapper functions based on execution mode
	var toolWrappers string
	if ce.Mode == ModeServer {
		toolWrappers = ce.generateGoToolWrappersServer()
	} else {
		toolWrappers = ce.generateGoToolWrappersDirect()
	}

	// Combine tool wrappers and user code
	fullScript := fmt.Sprintf(`
package main

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool wrapper functions
%s

func main() {
	ctx := context.Background()
	%s
}
`, toolWrappers, code)

	if err := os.WriteFile(scriptPath, []byte(fullScript), 0644); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	// Execute Go script
	execCtx, cancel := context.WithTimeout(ctx, ce.Timeout)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "go", "run", scriptPath)
	output, err := cmd.CombinedOutput()

	result := &ExecutionResult{
		Output: string(output),
		Stdout: string(output),
	}

	if err != nil {
		result.Error = err
	}

	return result, nil
}

// generatePythonToolWrappersServer creates Python wrapper functions for tools (server mode)
func (ce *CodeExecutor) generatePythonToolWrappersServer() string {
	var wrappers []string

	serverURL := ce.toolServer.GetBaseURL()

	// Create a mapping of tools that can be called via HTTP
	toolsMap := make(map[string]string)
	for _, tool := range ce.Tools {
		toolsMap[tool.Name()] = tool.Description()
	}

	// Serialize tools map for the wrapper
	toolsJSON, _ := json.Marshal(toolsMap)

	wrapper := fmt.Sprintf(`
# Available tools: %s
import json
try:
    import urllib.request
except ImportError:
    import urllib2 as urllib

TOOL_SERVER_URL = "%s"

def call_tool(tool_name, tool_input):
    """Call a tool through the HTTP tool server"""
    try:
        url = TOOL_SERVER_URL + "/call"
        data = json.dumps({
            "tool_name": tool_name,
            "input": tool_input
        }).encode('utf-8')

        req = urllib.request.Request(url, data=data, headers={'Content-Type': 'application/json'})
        response = urllib.request.urlopen(req)
        result = json.loads(response.read().decode('utf-8'))

        if result.get("success"):
            return result.get("result", "")
        else:
            return f"Error calling tool {tool_name}: {result.get('error', 'Unknown error')}"
    except Exception as e:
        return f"Error calling tool {tool_name}: {str(e)}"
`, string(toolsJSON), serverURL)

	wrappers = append(wrappers, wrapper)

	// Generate individual tool functions
	for _, tool := range ce.Tools {
		funcWrapper := fmt.Sprintf(`
def %s(input_data):
    """
    %s
    """
    return call_tool("%s", input_data)
`, sanitizeFunctionName(tool.Name()), tool.Description(), tool.Name())
		wrappers = append(wrappers, funcWrapper)
	}

	return strings.Join(wrappers, "\n")
}

// generatePythonToolWrappersDirect creates Python wrapper functions for tools (direct mode)
// In direct mode, tools are called directly through a helper process via stdin/stdout
func (ce *CodeExecutor) generatePythonToolWrappersDirect() string {
	// Create a helper program path
	helperPath := ce.createToolHelperProgram()

	var wrappers []string

	wrapper := fmt.Sprintf(`
# Direct tool execution (no HTTP server)
import subprocess
import json

TOOL_HELPER = "%s"

def call_tool_direct(tool_name, tool_input):
    """Call a tool directly through helper process"""
    try:
        request = json.dumps({
            "tool_name": tool_name,
            "input": str(tool_input)
        })

        result = subprocess.run(
            [TOOL_HELPER, request],
            capture_output=True,
            text=True,
            timeout=30
        )

        if result.returncode == 0:
            response = json.loads(result.stdout)
            if response.get("success"):
                return response.get("result", "")
            else:
                return f"Error: {response.get('error', 'Unknown error')}"
        else:
            return f"Error executing tool: {result.stderr}"
    except Exception as e:
        return f"Error calling tool {tool_name}: {str(e)}"
`, helperPath)

	wrappers = append(wrappers, wrapper)

	// Generate individual tool functions
	for _, tool := range ce.Tools {
		funcWrapper := fmt.Sprintf(`
def %s(input_data):
    """
    %s
    """
    return call_tool_direct("%s", input_data)
`, sanitizeFunctionName(tool.Name()), tool.Description(), tool.Name())
		wrappers = append(wrappers, funcWrapper)
	}

	return strings.Join(wrappers, "\n")
}

// generateGoToolWrappersServer creates Go wrapper functions for tools (server mode)
func (ce *CodeExecutor) generateGoToolWrappersServer() string {
	var wrappers []string

	serverURL := ce.toolServer.GetBaseURL()

	// Create the call_tool function
	wrapper := fmt.Sprintf(`
import (
	"bytes"
	"io"
	"net/http"
)

const toolServerURL = "%s"

// callTool calls a tool through the HTTP tool server
func callTool(ctx context.Context, toolName string, toolInput interface{}) (string, error) {
	requestBody := map[string]interface{}{
		"tool_name": toolName,
		"input":     toolInput,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %%w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", toolServerURL+"/call", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %%w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call tool: %%w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %%w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %%w", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		if resultStr, ok := result["result"].(string); ok {
			return resultStr, nil
		}
	}

	errorMsg := "unknown error"
	if errStr, ok := result["error"].(string); ok {
		errorMsg = errStr
	}
	return "", fmt.Errorf("tool execution failed: %%s", errorMsg)
}
`, serverURL)
	wrappers = append(wrappers, wrapper)

	// Generate individual tool functions
	for _, tool := range ce.Tools {
		funcWrapper := fmt.Sprintf(`
// %s: %s
func %s(ctx context.Context, input string) (string, error) {
	return callTool(ctx, "%s", input)
}
`, tool.Name(), tool.Description(), sanitizeFunctionName(tool.Name()), tool.Name())
		wrappers = append(wrappers, funcWrapper)
	}

	return strings.Join(wrappers, "\n")
}

// generateGoToolWrappersDirect creates Go wrapper functions for tools (direct mode)
func (ce *CodeExecutor) generateGoToolWrappersDirect() string {
	// Create helper program path
	helperPath := ce.createToolHelperProgram()

	var wrappers []string

	wrapper := fmt.Sprintf(`
import (
	"os/exec"
)

const toolHelper = "%s"

// callToolDirect calls a tool directly through helper process
func callToolDirect(ctx context.Context, toolName string, toolInput interface{}) (string, error) {
	request := map[string]interface{}{
		"tool_name": toolName,
		"input":     fmt.Sprintf("%%v", toolInput),
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %%w", err)
	}

	cmd := exec.CommandContext(ctx, toolHelper, string(jsonData))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute tool: %%w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %%w", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		if resultStr, ok := result["result"].(string); ok {
			return resultStr, nil
		}
	}

	errorMsg := "unknown error"
	if errStr, ok := result["error"].(string); ok {
		errorMsg = errStr
	}
	return "", fmt.Errorf("tool execution failed: %%s", errorMsg)
}
`, helperPath)

	wrappers = append(wrappers, wrapper)

	// Generate individual tool functions
	for _, tool := range ce.Tools {
		funcWrapper := fmt.Sprintf(`
// %s: %s
func %s(ctx context.Context, input string) (string, error) {
	return callToolDirect(ctx, "%s", input)
}
`, tool.Name(), tool.Description(), sanitizeFunctionName(tool.Name()), tool.Name())
		wrappers = append(wrappers, funcWrapper)
	}

	return strings.Join(wrappers, "\n")
}

// createToolHelperProgram creates a helper executable for direct tool execution
func (ce *CodeExecutor) createToolHelperProgram() string {
	// Create a temporary Go program that can execute tools
	helperPath := filepath.Join(ce.WorkDir, fmt.Sprintf("tool_helper_%d", time.Now().UnixNano()))

	// Generate Go source code for the helper
	helperSource := ce.generateHelperSource()

	sourcePath := helperPath + ".go"
	if err := os.WriteFile(sourcePath, []byte(helperSource), 0644); err != nil {
		// If we can't create the helper, return empty path
		// The calling code will handle the error
		return ""
	}

	// Compile the helper program
	cmd := exec.Command("go", "build", "-o", helperPath, sourcePath)
	if err := cmd.Run(); err != nil {
		// Compilation failed, clean up and return empty
		os.Remove(sourcePath)
		return ""
	}

	// Clean up source file
	os.Remove(sourcePath)

	return helperPath
}

// generateHelperSource generates the Go source code for the tool helper program
func (ce *CodeExecutor) generateHelperSource() string {
	// Build tool call implementations
	var toolCases []string
	for _, tool := range ce.Tools {
		toolCase := fmt.Sprintf(`	case "%s":
		result, err = tool_%s(ctx, req.Input)`,
			tool.Name(),
			sanitizeFunctionName(tool.Name()))
		toolCases = append(toolCases, toolCase)
	}

	// Build tool function implementations
	var toolFuncs []string
	for i, tool := range ce.Tools {
		// Capture tool index to avoid closure issues
		toolIndex := i
		_ = toolIndex

		toolFunc := fmt.Sprintf(`
func tool_%s(ctx context.Context, input string) (string, error) {
	// Direct execution of tool: %s
	// Note: In direct mode, we embed the tool logic here
	// For now, this is a placeholder that would need actual tool implementation
	return "Direct mode result for %s: " + input, nil
}`,
			sanitizeFunctionName(tool.Name()),
			tool.Description(),
			tool.Name())
		toolFuncs = append(toolFuncs, toolFunc)
	}

	source := fmt.Sprintf(`package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type Request struct {
	ToolName string ` + "`json:\"tool_name\"`" + `
	Input    string ` + "`json:\"input\"`" + `
}

type Response struct {
	Success bool   ` + "`json:\"success\"`" + `
	Result  string ` + "`json:\"result,omitempty\"`" + `
	Error   string ` + "`json:\"error,omitempty\"`" + `
}

%s

func main() {
	if len(os.Args) < 2 {
		respondError("missing request argument")
		return
	}

	var req Request
	if err := json.Unmarshal([]byte(os.Args[1]), &req); err != nil {
		respondError("invalid request: " + err.Error())
		return
	}

	ctx := context.Background()
	var result string
	var err error

	switch req.ToolName {
%s
	default:
		respondError("unknown tool: " + req.ToolName)
		return
	}

	if err != nil {
		respondError(err.Error())
		return
	}

	respondSuccess(result)
}

func respondSuccess(result string) {
	resp := Response{
		Success: true,
		Result:  result,
	}
	json.NewEncoder(os.Stdout).Encode(resp)
}

func respondError(errMsg string) {
	resp := Response{
		Success: false,
		Error:   errMsg,
	}
	json.NewEncoder(os.Stdout).Encode(resp)
}
`, strings.Join(toolFuncs, "\n"), strings.Join(toolCases, "\n"))

	return source
}

// sanitizeFunctionName converts a tool name to a valid function name
func sanitizeFunctionName(name string) string {
	// Replace invalid characters with underscores
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, ".", "_")

	// Ensure it starts with a letter
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		name = "tool_" + name
	}

	return name
}

// GetToolDefinitions returns tool definitions for LLM prompting
func (ce *CodeExecutor) GetToolDefinitions() string {
	var defs []string

	defs = append(defs, "# Available Tools\n")
	defs = append(defs, "You have access to the following tools that you can call in your code:\n")

	for _, tool := range ce.Tools {
		def := fmt.Sprintf("\n## %s\n", tool.Name())
		def += fmt.Sprintf("Description: %s\n", tool.Description())
		def += fmt.Sprintf("Usage: %s(input_string)\n", sanitizeFunctionName(tool.Name()))
		defs = append(defs, def)
	}

	return strings.Join(defs, "")
}
