package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/smallnest/goskills"
	adapter "github.com/smallnest/langgraphgo/adapter/goskills"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
	"github.com/smallnest/langgraphgo/llmtypes"
	"github.com/smallnest/langgraphgo/prebuilt"
	"github.com/smallnest/langgraphgo/tooltypes"
)

func main() {
	// 1. 初始化 LLM
	// 推荐使用 ERNIE 5.0 Thinking Preview，工具调用更稳定
	// 如需使用，设置环境变量：
	//   export OPENAI_API_KEY=your-ernie-api-key
	//   export OPENAI_BASE_URL=https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/ernie-5.0-thinking-preview
	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// 2. 从 skills 目录加载技能包
	skillsDir := "./skills"
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		skillsDir = "comic_skill_example/skills"
	}

	packages, err := goskills.ParseSkillPackages(skillsDir)
	if err != nil {
		log.Fatalf("解析技能包失败: %v", err)
	}

	if len(packages) == 0 {
		log.Fatal("在 " + skillsDir + " 中未找到任何技能")
	}

	// 3. 将技能转换为工具（工具配置会从 SKILL.md 自动读取）
	var allTools []tooltypes.Tool
	var allSystemMessages strings.Builder

	allSystemMessages.WriteString("你是一个有用的助手，可以访问工具来创作漫画。当用户要求创建漫画时，你必须调用 generate_comic_storyboard 函数。\n\n")
	allSystemMessages.WriteString("可用函数：\n")
	allSystemMessages.WriteString("- generate_comic_storyboard: 创建完整的漫画分镜脚本和提示词\n")
	allSystemMessages.WriteString("- generate_comic_image: 生成单张漫画图像（需要提示词和路径）\n")
	allSystemMessages.WriteString("- merge_comic_to_pdf: 将漫画图像合并成 PDF\n\n")
	allSystemMessages.WriteString("工作流程：\n")
	allSystemMessages.WriteString("1. 调用 generate_comic_storyboard 创建漫画分镜\n")
	allSystemMessages.WriteString("2. 如果输出显示 '=== IMAGE_GENERATION_REQUIRED ==='，则为每一页调用 generate_comic_image\n")
	allSystemMessages.WriteString("3. 调用 merge_comic_to_pdf 将所有图像合并成 PDF\n\n")
	allSystemMessages.WriteString("重要提示：始终调用函数，而不是提供文字描述。\n")

	for _, skill := range packages {
		fmt.Printf("正在加载技能: %s - %s\n", skill.Meta.Name, skill.Meta.Description)

		// 工具配置会从 SKILL.md 的 tools 字段自动读取
		// 如果需要覆盖，可以传入 ToolConfig
		skillTools, err := adapter.SkillsToTools(skill)
		if err != nil {
			log.Printf("转换技能 %s 为工具失败: %v", skill.Meta.Name, err)
			continue
		}

		allTools = append(allTools, skillTools...)

		for _, t := range skillTools {
			fmt.Printf("  - 工具: %s\n", t.Name())
		}
	}

	if len(allTools) == 0 {
		log.Fatal("未从技能中找到任何工具")
	}

	fmt.Printf("\n总共加载了 %d 个工具\n\n", len(allTools))

	// 4. 筛选出漫画相关工具
	var comicTools []tooltypes.Tool
	for _, t := range allTools {
		if t.Name() == "generate_comic_storyboard" || t.Name() == "generate_comic_image" || t.Name() == "merge_comic_to_pdf" {
			comicTools = append(comicTools, t)
		}
	}

	if len(comicTools) == 0 {
		log.Fatal("未找到漫画工具")
	}

	fmt.Printf("使用 %d 个漫画工具\n", len(comicTools))

	// 5. 调试：打印工具定义
	fmt.Println("\n=== 工具定义 ===")
	for _, t := range comicTools {
		fmt.Printf("工具: %s\n", t.Name())
		fmt.Printf("  描述: %s\n", t.Description())
		// 检查工具是否实现了 Schema
		if st, ok := t.(interface{ Schema() map[string]any }); ok {
			if schema := st.Schema(); schema != nil {
				fmt.Printf("  包含 Schema: 是\n")
			} else {
				fmt.Printf("  包含 Schema: 否（为 nil）\n")
			}
		} else {
			fmt.Printf("  包含 Schema: 否（接口不匹配）\n")
		}
	}
	fmt.Println("=== 工具定义结束 ===\n")

	// 6. 创建 Agent
	systemMsgStr := allSystemMessages.String()
	fmt.Printf("\n=== 系统消息 ===\n%s\n=== 系统消息结束 ===\n\n", systemMsgStr)

	// 设置是否禁用模型调用
	// 如果设置为 true，Agent 将跳过 LLM 调用，直接返回空响应
	// 这对于测试或仅执行工具调用时很有用
	disableModelInvocation := false

	agent, err := prebuilt.CreateAgentMap(llm, comicTools, 20,
		prebuilt.WithSystemMessage(systemMsgStr),
		prebuilt.WithDisableModelInvocation(disableModelInvocation),
	)
	if err != nil {
		log.Fatal(err)
	}

	// 7. 解析命令行参数
	if len(os.Args) < 2 {
		fmt.Println("用法: go run main.go <漫画描述>")
		fmt.Println()
		fmt.Println("示例:")
		fmt.Println("  go run main.go \"创作一个关于小姑娘在森林里采蘑菇的漫画\"")
		fmt.Println()
		fmt.Println("可用技能:")
		for _, skill := range packages {
			fmt.Printf("  - %s: %s\n", skill.Meta.Name, skill.Meta.Description)
		}
		os.Exit(1)
	}

	input := strings.Join(os.Args[1:], " ")

	// 8. 运行 Agent
	fmt.Printf("🎨 正在使用 Agent 创建漫画...\n")
	fmt.Printf("📝 请求: %s\n\n", input)

	ctx := context.Background()
	resp, err := agent.Invoke(ctx, map[string]any{
		"messages": []llmtypes.MessageContent{
			llmtypes.TextParts(llmtypes.ChatMessageTypeHuman, input),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 9. 打印结果
	fmt.Println("\n===========================================")
	fmt.Println("Agent 响应:")
	fmt.Println("===========================================")

	if messages, ok := resp["messages"].([]llmtypes.MessageContent); ok && len(messages) > 0 {
		for i, msg := range messages {
			fmt.Printf("\n[消息 %d - 角色: %s]\n", i+1, msg.Role)
			for j, part := range msg.Parts {
				switch p := part.(type) {
				case llmtypes.TextContent:
					fmt.Printf("  [部分 %d - 文本]: %s\n", j+1, string(p.Text))
				case llmtypes.ToolCall:
					fmt.Printf("  [部分 %d - 工具调用]: %s\n", j+1, p.FunctionCall.Name)
					fmt.Printf("    参数: %s\n", p.FunctionCall.Arguments)
				case llmtypes.ToolCallResponse:
					fmt.Printf("  [部分 %d - 工具响应]: %s\n", j+1, p)
				default:
					fmt.Printf("  [部分 %d - 未知类型]: %v\n", j+1, part)
				}
			}
		}
	} else {
		fmt.Printf("响应: %v\n", resp)
	}

	fmt.Println("\n===========================================")
	fmt.Println("完成!")
	fmt.Println("===========================================")
}
