package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/showcases/BettaFish/forum_engine"
	"github.com/smallnest/langgraphgo/showcases/BettaFish/insight_engine"
	"github.com/smallnest/langgraphgo/showcases/BettaFish/media_engine"
	"github.com/smallnest/langgraphgo/showcases/BettaFish/query_engine"
	"github.com/smallnest/langgraphgo/showcases/BettaFish/report_engine"
	"github.com/smallnest/langgraphgo/showcases/BettaFish/schema"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run main.go <查询>")
		return
	}

	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("错误: 未设置 OPENAI_API_KEY 环境变量。")
	}
	if os.Getenv("TAVILY_API_KEY") == "" {
		log.Fatal("错误: 未设置 TAVILY_API_KEY 环境变量。")
	}

	query := os.Args[1]

	// Initialize state
	initialState := schema.NewBettaFishState(query)

	// Create graph
	workflow := graph.NewStateGraph()

	// Add nodes
	workflow.AddNode("query_engine", query_engine.QueryEngineNode)
	workflow.AddNode("media_engine", media_engine.MediaEngineNode)
	workflow.AddNode("insight_engine", insight_engine.InsightEngineNode)
	workflow.AddNode("forum_engine", forum_engine.ForumEngineNode)
	workflow.AddNode("report_engine", report_engine.ReportEngineNode)

	// Add edges (Sequential for now)
	workflow.SetEntryPoint("query_engine")
	workflow.AddEdge("query_engine", "media_engine")
	workflow.AddEdge("media_engine", "insight_engine")
	workflow.AddEdge("insight_engine", "forum_engine")
	workflow.AddEdge("forum_engine", "report_engine")
	workflow.AddEdge("report_engine", graph.END)

	// Compile graph
	app, err := workflow.Compile()
	if err != nil {
		log.Fatalf("编译图失败: %v", err)
	}

	// Run graph
	ctx := context.Background()
	result, err := app.Invoke(ctx, initialState)
	if err != nil {
		log.Fatalf("运行图失败: %v", err)
	}

	// Print result
	finalState := result.(*schema.BettaFishState)
	fmt.Println("\n=== 执行完成 ===")
	fmt.Printf("报告已生成，包含 %d 个段落。\n", len(finalState.Paragraphs))
}
