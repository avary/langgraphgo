# Trading Agents - 交易代理系统

一个基于 LangGraph Go 和 LangChain Go 构建的多代理 LLM 驱动的金融交易框架。本项目是受 [TauricResearch/TradingAgents](https://github.com/TauricResearch/TradingAgents) 启发的 Go 语言实现。

## 概述

Trading Agents 模拟了一个专业的交易公司，由多个专业化的 AI 代理协作分析市场并做出明智的交易决策。

## 架构

### 代理团队

1. **分析师团队**
   - **基本面分析师**: 评估公司财务状况和业绩指标
   - **情绪分析师**: 分析社交媒体和公众情绪
   - **新闻分析师**: 监控全球新闻和宏观经济指标
   - **技术分析师**: 使用技术指标进行价格趋势分析

2. **研究团队**
   - **看涨研究员**: 寻找买入机会
   - **看跌研究员**: 识别潜在风险和卖出信号

3. **交易员**
   - 综合所有分析师和研究员的报告
   - 做出最终交易决策

4. **风险管理团队**
   - 监控投资组合风险敞口
   - 实施风险缓解策略
   - 确保符合风险承受能力

## 功能特性

- ✅ 多代理协作分析
- ✅ 实时市场数据集成
- ✅ 后端 API 服务器
- ✅ 命令行界面 (CLI)
- ✅ Web 仪表板
- ✅ 完善的日志和追踪功能

## 组件结构

```
trading_agents/
├── backend/        # API 服务器实现
├── cli/            # 命令行界面
├── web/            # Web 前端
├── agents/         # 代理实现
├── tools/          # 市场数据和分析工具
└── config/         # 配置文件
```

## 快速开始

### 前置要求

- Go 1.21+
- OpenAI API 密钥 (用于 LLM)
- Alpha Vantage API 密钥 (用于市场数据)

### 安装

```bash
# 设置环境变量
export OPENAI_API_KEY="your-openai-key"
export ALPHA_VANTAGE_API_KEY="your-alpha-vantage-key"

# 构建所有组件
go build -o bin/trading-agents ./showcases/trading_agents/backend
go build -o bin/trading-cli ./showcases/trading_agents/cli
```

### 运行后端服务

```bash
./bin/trading-agents --port 8080
```

### 运行 CLI

```bash
# 分析股票
./bin/trading-cli -cmd analyze -symbol AAPL

# 获取交易建议
./bin/trading-cli -cmd recommend -symbol AAPL -capital 10000
```

### 运行 Web 界面

```bash
# 首先启动后端，然后
cd showcases/trading_agents/web
# 在浏览器中打开 index.html
```

## 使用示例

### 后端 API

```bash
# 健康检查
curl http://localhost:8080/health

# 分析股票
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{"symbol": "AAPL", "timeframe": "1D"}'

# 获取交易建议
curl -X POST http://localhost:8080/api/recommend \
  -H "Content-Type: application/json" \
  -d '{"symbol": "AAPL", "capital": 10000, "risk_tolerance": "moderate"}'
```

### CLI 命令行

```bash
# 快速分析
./bin/trading-cli -cmd analyze -symbol TSLA --verbose

# 详细建议和风险评估
./bin/trading-cli -cmd recommend -symbol GOOGL -capital 50000 -risk-level low

# 监控多只股票
./bin/trading-cli -cmd watch -symbols AAPL,GOOGL,TSLA --interval 5m
```

## 免责声明

⚠️ **本框架仅用于研究和教育目的。它不构成任何金融、投资或交易建议。在做出投资决策之前，请务必咨询合格的金融专业人士。**

## 参考资料

- 原始项目: [TauricResearch/TradingAgents](https://github.com/TauricResearch/TradingAgents)
- 论文: [arXiv:2412.20138](https://arxiv.org/abs/2412.20138)
- LangGraph Go: [smallnest/langgraphgo](https://github.com/smallnest/langgraphgo)
- LangChain Go: [tmc/langchaingo](https://github.com/tmc/langchaingo)

## 许可证

MIT License - 详见 LICENSE 文件
