package goskills

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/smallnest/goskills"
	goskillstool "github.com/smallnest/goskills/tool"
	"github.com/smallnest/langgraphgo/tool"
)

// ToolConfig 定义工具的配置，可以从 SKILL.md 或单独的配置文件中读取
// 注意：如果 SKILL.md 中已经定义了 tools 字段，此配置将被用作补充覆盖
type ToolConfig struct {
	// NameMapping 定义工具名称映射，从脚本名称到更友好的名称
	NameMapping map[string]string `json:"nameMapping"`

	// SchemaOverrides 定义工具 schema 的覆盖
	SchemaOverrides map[string]map[string]any `json:"schemaOverrides"`

	// DescriptionOverrides 定义工具描述的覆盖
	DescriptionOverrides map[string]string `json:"descriptionOverrides"`
}

// buildToolConfigFromSkill 从 SKILL.md 中的工具定义自动构建 ToolConfig
func buildToolConfigFromSkill(skill *goskills.SkillPackage) *ToolConfig {
	if len(skill.Meta.Tools) == 0 {
		return nil
	}

	config := &ToolConfig{
		NameMapping:          make(map[string]string),
		DescriptionOverrides: make(map[string]string),
		SchemaOverrides:      make(map[string]map[string]any),
	}

	for _, toolDef := range skill.Meta.Tools {
		// 构建名称映射：从工具名到工具名（保持一致）
		config.NameMapping[toolDef.Name] = toolDef.Name

		// 设置描述
		if toolDef.Description != "" {
			config.DescriptionOverrides[toolDef.Name] = toolDef.Description
		}

		// 构建 schema
		schema := map[string]any{
			"type":       "object",
			"properties": make(map[string]any),
		}

		if len(toolDef.Parameters) > 0 {
			var required []string
			for paramName, param := range toolDef.Parameters {
				prop := map[string]any{
					"type": param.Type,
				}
				if param.Description != "" {
					prop["description"] = param.Description
				}
				schema["properties"].(map[string]any)[paramName] = prop
				if param.Required {
					required = append(required, paramName)
				}
			}
			if len(required) > 0 {
				schema["required"] = required
			}
		} else {
			// 默认参数: args 数组
			schema["properties"] = map[string]any{
				"args": map[string]any{
					"type":        "array",
					"description": "Arguments to pass to the script.",
					"items": map[string]any{
						"type": "string",
					},
				},
			}
		}

		schema["additionalProperties"] = false
		config.SchemaOverrides[toolDef.Name] = schema
	}

	return config
}

// SkillTool implements tool.Tool for goskills.
type SkillTool struct {
	name        string
	description string
	scriptMap   map[string]string
	skillPath   string
	config      *ToolConfig            // 工具配置
	schema      map[string]any         // 工具的 JSON schema
	skill       *goskills.SkillPackage // 保留对 skill 包的引用以获取工具定义
}

var _ tool.Tool = &SkillTool{}
var _ interface{ Schema() map[string]any } = &SkillTool{}

func (t *SkillTool) Name() string {
	// 如果配置中有名称映射，使用映射后的名称
	if t.config != nil && t.config.NameMapping != nil {
		if newName, ok := t.config.NameMapping[t.name]; ok {
			return newName
		}
	}
	return t.name
}

func (t *SkillTool) Description() string {
	// 使用映射后的名称获取覆盖的描述
	if t.config != nil && t.config.DescriptionOverrides != nil {
		mappedName := t.Name()
		if desc, ok := t.config.DescriptionOverrides[mappedName]; ok {
			return desc
		}
	}
	return t.description
}

func (t *SkillTool) Schema() map[string]any {
	// 如果配置中有 schema 覆盖，使用覆盖的 schema
	if t.config != nil && t.config.SchemaOverrides != nil {
		mappedName := t.Name()
		if schema, ok := t.config.SchemaOverrides[mappedName]; ok {
			return schema
		}
	}
	return t.schema
}

// getToolDefinition 获取当前工具的定义
func (t *SkillTool) getToolDefinition() *goskills.ToolDefinition {
	if t.skill == nil || len(t.skill.Meta.Tools) == 0 {
		return nil
	}
	for i := range t.skill.Meta.Tools {
		if t.skill.Meta.Tools[i].Name == t.name {
			return &t.skill.Meta.Tools[i]
		}
	}
	return nil
}

// buildParamMapping 动态构建参数名到 CLI flag 的映射
// 规则：将参数名转换为 kebab-case 并添加 "--" 前缀
// 例如: "filePath" -> "--file-path", "quality" -> "--quality"
func (t *SkillTool) buildParamMapping(toolDef *goskills.ToolDefinition) map[string]string {
	if toolDef == nil || len(toolDef.Parameters) == 0 {
		return nil
	}

	mapping := make(map[string]string)
	for paramName := range toolDef.Parameters {
		flag := toCLIFlag(paramName)
		mapping[paramName] = flag
	}
	return mapping
}

// toCLIFlag 将参数名转换为 CLI flag 格式
// 例如: "filePath" -> "--file-path", "quality" -> "--quality", "path" -> "--path"
func toCLIFlag(paramName string) string {
	var result []rune
	for i, r := range paramName {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '-')
		}
		result = append(result, r)
	}
	return "--" + string(result)
}

func (t *SkillTool) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"name":        t.name,
		"description": t.description,
		"skillPath":   t.skillPath,
		"scriptMap":   t.scriptMap,
		"mappedName":  t.Name(), // 映射后的名称
	})
}

func (t *SkillTool) Call(ctx context.Context, input string) (string, error) {
	// 使用原始名称进行路由，因为脚本路径是用原始名称存储的
	originalName := t.name

	switch originalName {
	case "run_shell_code":
		var params struct {
			Code string         `json:"code"`
			Args map[string]any `json:"args"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_shell_code arguments: %w", err)
		}
		shellTool := goskillstool.ShellTool{}
		return shellTool.Run(params.Args, params.Code)

	case "run_shell_script":
		var params struct {
			ScriptPath string   `json:"scriptPath"`
			Args       []string `json:"args"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_shell_script arguments: %w", err)
		}
		return tool.RunShellScript(params.ScriptPath, params.Args)

	case "run_python_code":
		var params struct {
			Code string         `json:"code"`
			Args map[string]any `json:"args"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_python_code arguments: %w", err)
		}
		pythonTool := goskillstool.PythonTool{}
		return pythonTool.Run(params.Args, params.Code)

	case "run_python_script":
		var params struct {
			ScriptPath string   `json:"scriptPath"`
			Args       []string `json:"args"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal run_python_script arguments: %w", err)
		}
		return goskillstool.RunPythonScript(params.ScriptPath, params.Args)

	case "read_file":
		var params struct {
			FilePath string `json:"filePath"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal read_file arguments: %w", err)
		}
		path := params.FilePath
		if !filepath.IsAbs(path) && t.skillPath != "" {
			resolvedPath := filepath.Join(t.skillPath, path)
			if _, err := os.Stat(resolvedPath); err == nil {
				path = resolvedPath
			}
		}
		return goskillstool.ReadFile(path)

	case "write_file":
		var params struct {
			FilePath string `json:"filePath"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal write_file arguments: %w", err)
		}
		err := goskillstool.WriteFile(params.FilePath, params.Content)
		if err == nil {
			return fmt.Sprintf("Successfully wrote to file: %s", params.FilePath), nil
		}
		return "", err

	case "wikipedia_search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal wikipedia_search arguments: %w", err)
		}
		return goskillstool.WikipediaSearch(params.Query)

	case "tavily_search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal tavily_search arguments: %w", err)
		}
		return goskillstool.TavilySearch(params.Query)

	case "web_fetch":
		var params struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal([]byte(input), &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal web_fetch arguments: %w", err)
		}
		return goskillstool.WebFetch(params.URL)

	default:
		if scriptPath, ok := t.scriptMap[originalName]; ok {
			// 尝试解析为命名参数格式（来自 SKILL.md 工具定义）
			var namedParams map[string]any
			err := json.Unmarshal([]byte(input), &namedParams)

			if err == nil && len(namedParams) > 0 {
				// 成功解析为命名参数，转换为命令行参数
				var args []string

				// 获取工具定义并动态构建参数映射
				toolDef := t.getToolDefinition()
				paramMapping := t.buildParamMapping(toolDef)

				// 如果没有工具定义，使用原始参数名作为 flag（添加 -- 前缀）
				if paramMapping == nil {
					for key := range namedParams {
						paramMapping = map[string]string{key: toCLIFlag(key)}
					}
				}

				// 构建命令行参数
				for key, value := range namedParams {
					if value != nil {
						// 跳过 "args" 键，这是为了向后兼容旧的数组格式
						if key == "args" {
							continue
						}
						flag, ok := paramMapping[key]
						if !ok {
							// 如果映射中没有，动态生成
							flag = toCLIFlag(key)
						}
						args = append(args, flag)
						args = append(args, fmt.Sprintf("%v", value))
					}
				}

				if strings.HasSuffix(scriptPath, ".py") {
					return goskillstool.RunPythonScript(scriptPath, args)
				} else if strings.HasSuffix(scriptPath, ".ts") || strings.HasSuffix(scriptPath, ".js") {
					return tool.RunTypeScriptScript(scriptPath, args)
				} else {
					return tool.RunShellScript(scriptPath, args)
				}
			}

			// 回退到旧的 args 数组格式
			var params struct {
				Args []string `json:"args"`
			}
			if input != "" {
				if err := json.Unmarshal([]byte(input), &params); err != nil {
					return "", fmt.Errorf("failed to unmarshal script arguments: %w", err)
				}
			}
			if strings.HasSuffix(scriptPath, ".py") {
				return goskillstool.RunPythonScript(scriptPath, params.Args)
			} else if strings.HasSuffix(scriptPath, ".ts") || strings.HasSuffix(scriptPath, ".js") {
				return tool.RunTypeScriptScript(scriptPath, params.Args)
			} else {
				return tool.RunShellScript(scriptPath, params.Args)
			}
		}
		return "", fmt.Errorf("unknown tool: %s", originalName)
	}
}

// SkillsToToolsOptions 定义转换选项
type SkillsToToolsOptions struct {
	// ToolConfig 提供工具配置（名称映射、schema 覆盖等）
	// 注意：如果 SKILL.md 中已经定义了 tools，会自动生成配置，
	//      此配置仅用于覆盖或补充 SKILL.md 中的定义
	ToolConfig *ToolConfig
}

// SkillsToTools converts a goskills.SkillPackage to a slice of tool.Tool.
// 自动从 SKILL.md 读取工具定义，如果没有定义则自动生成。
// 支持通过 ToolConfig 覆盖或补充 SKILL.md 中的定义。
func SkillsToTools(skill *goskills.SkillPackage, opts ...SkillsToToolsOptions) ([]tool.Tool, error) {
	var config *ToolConfig

	// 1. 首先尝试从 SKILL.md 自动构建配置
	skillConfig := buildToolConfigFromSkill(skill)
	if skillConfig != nil {
		config = skillConfig
	}

	// 2. 如果用户提供了配置，合并覆盖
	if len(opts) > 0 && opts[0].ToolConfig != nil {
		if config == nil {
			config = &ToolConfig{
				NameMapping:          make(map[string]string),
				DescriptionOverrides: make(map[string]string),
				SchemaOverrides:      make(map[string]map[string]any),
			}
		}
		// 合并 NameMapping
		maps.Copy(config.NameMapping, opts[0].ToolConfig.NameMapping)
		// 合并 DescriptionOverrides
		maps.Copy(config.DescriptionOverrides, opts[0].ToolConfig.DescriptionOverrides)
		// 合并 SchemaOverrides
		maps.Copy(config.SchemaOverrides, opts[0].ToolConfig.SchemaOverrides)
	}

	availableTools, scriptMap := goskills.GenerateToolDefinitions(skill)
	var result []tool.Tool

	for _, t := range availableTools {
		if t.Function.Name == "" {
			continue
		}

		// 创建描述，如果可用的话包含参数 schema
		desc := t.Function.Description

		// 从函数参数构建默认 schema
		var schema map[string]any
		if t.Function.Parameters != nil {
			// 尝试类型断言为 map[string]any
			if params, ok := t.Function.Parameters.(map[string]any); ok {
				// Parameters 是一个完整的 schema 对象，包含 type, properties, required 等
				// 直接使用它作为 schema，但确保包含 additionalProperties
				schema = params
				if _, exists := schema["additionalProperties"]; !exists {
					schema["additionalProperties"] = false
				}
			}
		}

		result = append(result, &SkillTool{
			name:        t.Function.Name,
			description: desc,
			scriptMap:   scriptMap,
			skillPath:   skill.Path,
			config:      config,
			schema:      schema,
			skill:       skill,
		})
	}

	return result, nil
}

// MCPToTools converts MCP tools to langchaingo tools.
// Note: goskills also supports MCP. We can add a helper for that too if needed,
// but the user specifically asked for "Skills封装".
