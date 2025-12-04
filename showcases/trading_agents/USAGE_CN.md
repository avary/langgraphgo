# Trading Agents 使用指南

## 快速开始

### 1. 设置环境

```bash
# 设置 API 密钥
export OPENAI_API_KEY="your-openai-api-key"
export ALPHA_VANTAGE_API_KEY="your-alpha-vantage-key"  # 可选
```

### 2. 构建应用程序

```bash
# 从 langgraphgo 根目录
cd showcases/trading_agents

# 构建后端服务器
go build -o ../../bin/trading-agents ./backend

# 构建 CLI 工具
go build -o ../../bin/trading-cli ./cli
```

## 使用后端 API

### 启动服务器

```bash
./bin/trading-agents --port 8080
```

服务器将在 `http://localhost:8080` 启动

### API 端点

#### 健康检查
```bash
curl http://localhost:8080/health
```

#### 完整分析
```bash
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "AAPL",
    "capital": 10000,
    "risk_tolerance": "moderate",
    "timeframe": "1D"
  }'
```

#### 快速检查
```bash
curl -X POST http://localhost:8080/api/quick-check \
  -H "Content-Type: application/json" \
  -d '{"symbol": "TSLA"}'
```

## 使用 CLI 命令行工具

### 完整分析
```bash
./bin/trading-cli -cmd analyze -symbol AAPL -verbose
```

### 交易建议
```bash
./bin/trading-cli -cmd recommend -symbol GOOGL -capital 50000 -risk-level low
```

### 快速检查
```bash
./bin/trading-cli -cmd quick -symbol TSLA
```

### CLI 选项

- `-cmd` : 命令 (analyze, recommend, quick)
- `-symbol` : 股票代码 (必需)
- `-capital` : 可用资金（美元）(默认: 10000)
- `-risk-level` : 风险承受能力 (low, moderate, high)
- `-timeframe` : 交易时间框架 (5m, 1H, 1D, 1W)
- `-verbose` : 显示详细的代理报告
- `-json` : 以 JSON 格式输出

## 使用 Web 界面

### 1. 启动后端服务器

```bash
./bin/trading-agents --port 8080
```

### 2. 打开 Web 界面

直接在浏览器中打开 `web/index.html` 文件:

```bash
open showcases/trading_agents/web/index.html
```

或者使用简单的 HTTP 服务器:

```bash
cd showcases/trading_agents/web
python3 -m http.server 3000
# 然后在浏览器中打开 http://localhost:3000
```

### 3. 分析股票

1. 输入股票代码（例如：AAPL, GOOGL, TSLA）
2. 设置资金和风险承受能力
3. 点击"分析股票"
4. 查看多代理分析和交易建议

## 示例工作流

### 日内交易工作流

```bash
# 对多只股票进行快速检查
./bin/trading-cli -cmd quick -symbol AAPL
./bin/trading-cli -cmd quick -symbol GOOGL
./bin/trading-cli -cmd quick -symbol TSLA

# 对最佳候选进行完整分析
./bin/trading-cli -cmd analyze -symbol AAPL -timeframe 5m -verbose
```

### 投资分析工作流

```bash
# 保守的长期投资
./bin/trading-cli -cmd recommend \
  -symbol MSFT \
  -capital 100000 \
  -risk-level low \
  -timeframe 1W \
  -verbose
```

### API 集成示例

```python
import requests

# 分析股票
response = requests.post('http://localhost:8080/api/analyze', json={
    'symbol': 'AAPL',
    'capital': 50000,
    'risk_tolerance': 'moderate'
})

result = response.json()
print(f"建议: {result['recommendation']}")
print(f"置信度: {result['confidence']}%")
print(f"持仓规模: {result['position_size']} 股")
```

## 理解输出结果

### 建议类型

- **BUY** 🟢: 识别到强烈的买入机会
- **SELL** 🔴: 卖出建议或做空机会
- **HOLD** 🟡: 维持当前持仓或观望

### 置信度分数

- **80-100%**: 非常高的置信度，所有代理都发出强烈信号
- **60-80%**: 良好置信度，大多数代理意见一致
- **40-60%**: 中等置信度，信号混合
- **低于 40%**: 低置信度，信号冲突

### 风险评分

- **0-30**: 低风险，保守交易
- **30-70**: 中等风险，平衡方法
- **70-100**: 高风险，激进交易

### 代理报告

每次分析包括以下代理的报告:

1. **基本面分析师**: 公司财务和估值
2. **情绪分析师**: 社交媒体和公众情绪
3. **技术分析师**: 图表模式和指标
4. **看涨研究员**: 积极观点和机会
5. **看跌研究员**: 风险和警示信号
6. **风险管理员**: 风险评估和缓解策略

**交易员**综合所有报告做出最终建议。

## 故障排除

### "API key is required" 错误

确保已设置 OpenAI API 密钥:
```bash
export OPENAI_API_KEY="your-key-here"
```

### "Analysis failed" 错误

- 检查互联网连接
- 验证股票代码是否有效
- 确保后端服务器正在运行（针对 Web 界面）

### 后端服务器问题

检查服务器是否运行:
```bash
curl http://localhost:8080/health
```

查看服务器日志进行调试:
```bash
./bin/trading-agents --verbose
```

## 最佳实践建议

1. **使用有效代码**: 确保使用正确的股票代码（例如：AAPL 代表苹果，而不是 APPLE）

2. **设置实际资金**: 使用实际资金金额以获得准确的持仓规模

3. **匹配风险承受能力**: 选择与您实际风险承受能力相匹配的风险等级

4. **查看所有报告**: 不要只看建议 - 阅读详细分析

5. **考虑上下文**: 分析是时点性的。市场条件变化迅速。

6. **结合研究**: 将此作为研究过程中的众多工具之一

## 免责声明

⚠️ **重要**: 本工具仅用于**教育和研究目的**。

- 不构成金融建议
- 不构成投资建议
- 务必咨询合格的金融专业人士
- 过去的业绩不保证未来结果
- 您需对自己的投资决策负责

## 获取帮助

- 报告问题: [GitHub Issues](https://github.com/smallnest/langgraphgo/issues)
- 文档: 参见 README.md
- 示例: 查看 `examples/` 目录

## 下一步

- 尝试分析不同的股票
- 尝试不同的风险承受水平
- 比较不同时间框架的建议
- 使用 API 构建您自己的交易策略
