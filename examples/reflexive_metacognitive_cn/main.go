// Reflexive Metacognitive Agent (Chinese Version)
// 这是一个实现 Fareed Khan 的 Agentic Architectures 系列中的“反思性元认知 Agent”架构的示例。

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/llms"
	openai "github.com/smallnest/langgraphgo/llms/nativeopenai"
)

// ==================== 数据模型 ====================

type AgentSelfModel struct {
	Name                string
	Role                string
	KnowledgeDomain     []string
	AvailableTools      []string
	ConfidenceThreshold float64
}

type MetacognitiveAnalysis struct {
	Confidence float64
	Strategy   string
	Reasoning  string
	ToolToUse  string
	ToolArgs   map[string]string
}

type AgentState struct {
	UserQuery             string
	SelfModel             *AgentSelfModel
	MetacognitiveAnalysis *MetacognitiveAnalysis
	ToolOutput            string
	FinalResponse         string
}

// ==================== 工具 ====================

type DrugInteractionChecker struct {
	knownInteractions map[string]string
}

func (d *DrugInteractionChecker) Check(drugA, drugB string) string {
	key := drugA + "+" + drugB
	if interaction, ok := d.knownInteractions[key]; ok {
		return fmt.Sprintf("发现相互作用：%s", interaction)
	}
	return "未发现明显的相互作用。但请务必咨询医生。"
}

func NewDrugInteractionChecker() *DrugInteractionChecker {
	return &DrugInteractionChecker{
		knownInteractions: map[string]string{
			"布洛芬+利辛诺普利": "中度风险：布洛芬可能会降低利辛诺普利的降压效果。请监测血压。",
			"阿司匹林+华法林":  "高风险：增加出血风险。除非医生指导，否则应避免这种组合。",
		},
	}
}

var drugTool = NewDrugInteractionChecker()

// ==================== 图节点 ====================

func MetacognitiveAnalysisNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	agentState := state["agent_state"].(*AgentState)

	fmt.Println("\n--- 🤔 Agent 正在进行元认知分析... ---")

	prompt := fmt.Sprintf(`你是 AI 助手的元认知推理引擎。你的任务是根据 Agent 的自我模型分析用户的查询。

**Agent 自我模型：**
- 名称：%s
- 角色：%s
- 知识领域：%s
- 可用工具：%s

**策略规则：**
1. **escalate (上报)**：涉及紧急情况、不在知识领域内或有任何疑虑。
2. **use_tool (使用工具)**：需要使用 'drug_interaction_checker'。
3. **reason_directly (直接回答)**：在知识领域内且风险较低。

格式：
CONFIDENCE: [0.0 到 1.0]
STRATEGY: [escalate|use_tool|reason_directly]
TOOL_TO_USE: [工具名称或 "none"]
DRUG_A: [药物 A 名称或 "none"]
DRUG_B: [药物 B 名称 or "none"]
REASONING: [简要理由]

**用户查询：** %s`,
		agentState.SelfModel.Name,
		agentState.SelfModel.Role,
		strings.Join(agentState.SelfModel.KnowledgeDomain, ", "),
		strings.Join(agentState.SelfModel.AvailableTools, ", "),
		agentState.UserQuery)

	llm := state["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, fmt.Errorf("元认知分析调用失败: %w", err)
	}

	analysis := parseMetacognitiveAnalysis(resp)
	agentState.MetacognitiveAnalysis = analysis

	fmt.Printf("置信度: %.2f | 策略: %s\n", analysis.Confidence, analysis.Strategy)
	return state, nil
}

func ReasonDirectlyNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	agentState := state["agent_state"].(*AgentState)
	fmt.Println("--- ✅ 直接回答中... ---")

	prompt := fmt.Sprintf(`你是 %s。请提供一个有用的、非处方性的回答。提醒：你不是医生。

查询：%s`, agentState.SelfModel.Role, agentState.UserQuery)

	llm := state["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, err
	}

	agentState.FinalResponse = resp
	return state, nil
}

func CallToolNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	agentState := state["agent_state"].(*AgentState)
	fmt.Printf("--- 🛠️  调用工具 `%s`... ---\n", agentState.MetacognitiveAnalysis.ToolToUse)

	analysis := agentState.MetacognitiveAnalysis
	if analysis.ToolToUse == "drug_interaction_checker" {
		agentState.ToolOutput = drugTool.Check(analysis.ToolArgs["drug_a"], analysis.ToolArgs["drug_b"])
	} else {
		agentState.ToolOutput = "错误：未找到工具。"
	}
	return state, nil
}

func SynthesizeToolResponseNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	agentState := state["agent_state"].(*AgentState)
	fmt.Println("--- 📝 综合工具输出... ---")

	prompt := fmt.Sprintf(`你是 %s。请结合工具输出向用户提供帮助。务必声明你不是医生。

查询：%s
工具输出：%s`, agentState.SelfModel.Role, agentState.UserQuery, agentState.ToolOutput)

	llm := state["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, err
	}

	agentState.FinalResponse = resp
	return state, nil
}

func EscalateToHumanNode(ctx context.Context, state map[string]any) (map[string]any, error) {
	agentState := state["agent_state"].(*AgentState)
	fmt.Println("--- 🚨 风险较高，正在上报... ---")

	agentState.FinalResponse = "我是 AI 助手，不具备提供此话题相关信息的资质。**请立即咨询医疗专业人员。**"
	return state, nil
}

func RouteStrategy(ctx context.Context, state map[string]any) string {
	agentState := state["agent_state"].(*AgentState)
	switch agentState.MetacognitiveAnalysis.Strategy {
	case "reason_directly":
		return "reason"
	case "use_tool":
		return "call_tool"
	default:
		return "escalate"
	}
}

func parseMetacognitiveAnalysis(response string) *MetacognitiveAnalysis {
	analysis := &MetacognitiveAnalysis{Confidence: 0.1, Strategy: "escalate", ToolArgs: make(map[string]string)}
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		key := strings.ToUpper(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])
		switch key {
		case "CONFIDENCE":
			fmt.Sscanf(val, "%f", &analysis.Confidence)
		case "STRATEGY":
			analysis.Strategy = strings.ToLower(val)
		case "TOOL_TO_USE":
			analysis.ToolToUse = strings.ToLower(val)
		case "DRUG_A":
			analysis.ToolArgs["drug_a"] = val
		case "DRUG_B":
			analysis.ToolArgs["drug_b"] = val
		case "REASONING":
			analysis.Reasoning = val
		}
	}
	return analysis
}

func main() {
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("请设置 OPENAI_API_KEY")
	}

	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	medicalAgentModel := &AgentSelfModel{
		Name:                "分诊机器人-3000",
		Role:                "提供初步医疗信息的 AI 助手",
		KnowledgeDomain:     []string{"感冒", "流感", "过敏", "头痛", "急救"},
		AvailableTools:      []string{"drug_interaction_checker"},
		ConfidenceThreshold: 0.6,
	}

	workflow := graph.NewStateGraph[map[string]any]()
	workflow.AddNode("analyze", "元认知分析", MetacognitiveAnalysisNode)
	workflow.AddNode("reason", "直接回答", ReasonDirectlyNode)
	workflow.AddNode("call_tool", "调用工具", CallToolNode)
	workflow.AddNode("synthesize", "综合输出", SynthesizeToolResponseNode)
	workflow.AddNode("escalate", "上报", EscalateToHumanNode)

	workflow.SetEntryPoint("analyze")
	workflow.AddConditionalEdge("analyze", RouteStrategy)
	workflow.AddEdge("reason", graph.END)
	workflow.AddEdge("call_tool", "synthesize")
	workflow.AddEdge("synthesize", graph.END)
	workflow.AddEdge("escalate", graph.END)

	app, err := workflow.Compile()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("--- 测试：查询感冒症状 ---")
	agentState := &AgentState{UserQuery: "感冒有哪些症状？", SelfModel: medicalAgentModel}
	result, _ := app.Invoke(context.Background(), map[string]any{"llm": llm, "agent_state": agentState})
	fmt.Printf("\n回答：%s\n", result["agent_state"].(*AgentState).FinalResponse)
}
