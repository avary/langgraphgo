# BettaFish (Go Implementation)

This is a **complete replication** of the [BettaFish](https://github.com/666ghj/BettaFish) project in Go, using [langgraphgo](https://github.com/smallnest/langgraphgo) and [langchaingo](https://github.com/tmc/langchaingo).

It implements the full multi-agent architecture for deep public opinion analysis.

## Features

- **QueryEngine**: 
  - Generates a structured research plan (outline).
  - Performs deep web search using Tavily API.
  - Implements a **Reflection Loop** to iteratively refine search results and summaries.
  - Uses specialized prompts for searching, summarizing, and reflecting.
- **MediaEngine**: 
  - Searches for relevant images using Tavily's image search capabilities.
- **InsightEngine**: 
  - (Simulated) Mines internal data for insights.
- **ForumEngine**: 
  - Facilitates an LLM-driven discussion between "NewsAgent", "MediaAgent", and "Moderator" to synthesize findings.
- **ReportEngine**: 
  - Compiles all findings into a comprehensive Markdown report.

## Prerequisites

You need the following API keys:
- `OPENAI_API_KEY`: For LLM inference (GPT-4o recommended).
- `TAVILY_API_KEY`: For web search and image search.

## Usage

```bash
export OPENAI_API_KEY="sk-..."
export TAVILY_API_KEY="tvly-..."
go run showcases/BettaFish/main.go "Your Research Topic"
```
