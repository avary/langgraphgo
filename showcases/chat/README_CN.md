# LangGraphGo 聊天应用

基于Web的多会话聊天应用，支持持久化本地历史记录。支持OpenAI和OpenAI兼容API（百度、Azure、本地模型等）。

## 特性

- 🔄 **多会话支持**：创建和管理多个独立的聊天会话
- 💾 **持久化历史**：所有对话自动保存到本地磁盘
- 🌐 **Web界面**：简洁现代的Web用户界面
- 🤖 **智能聊天Agent**：轻量级聊天agent，带会话历史管理
- 🔌 **多提供商支持**：支持OpenAI、百度、Azure和任何OpenAI兼容API
- 🎨 **精美UI**：响应式设计，流畅动画
- 📝 **会话管理**：创建、查看和删除会话
- ⏱️ **实时更新**：查看消息数量和时间戳

## 快速开始

### 1. 配置

```bash
cd showcases/chat

# 复制环境变量模板
cp .env.example .env

# 编辑.env并添加你的API密钥
# OPENAI_API_KEY=your-key-here
```

### 2. 安装依赖

```bash
go mod tidy
```

### 3. 运行服务器

```bash
# 使用便捷脚本
./start.sh

# 或手动运行
go run main.go session.go
```

服务器将在 `http://localhost:8080` 启动

### 4. 使用Web界面

1. 在浏览器中打开 `http://localhost:8080`
2. 如果没有会话，系统会自动创建一个
3. 立即开始聊天！

## 配置

环境变量（在`.env`中）：

```env
# 必需：你的API密钥
OPENAI_API_KEY=your-api-key-here

# 可选：模型名称（默认：gpt-4o-mini）
OPENAI_MODEL=gpt-4o-mini

# 可选：OpenAI兼容API的Base URL
# 留空表示使用标准OpenAI API
# 示例：
#   百度：https://your-baidu-endpoint/v1
#   Azure：https://your-resource.openai.azure.com/
#   Ollama：http://localhost:11434/v1
OPENAI_BASE_URL=

# 可选：服务器端口（默认：8080）
PORT=8080

# 可选：会话存储目录（默认：./sessions）
SESSION_DIR=./sessions

# 可选：每个会话最大消息数（默认：50）
MAX_HISTORY_SIZE=50
```

### 使用不同的LLM提供商

**OpenAI（默认）**：
```env
OPENAI_API_KEY=sk-your-openai-key
```

**百度千帆**：
```env
OPENAI_API_KEY=your-baidu-token
OPENAI_BASE_URL=https://your-baidu-endpoint/v1
OPENAI_MODEL=your-model-name
```

**Azure OpenAI**：
```env
OPENAI_API_KEY=your-azure-key
OPENAI_BASE_URL=https://your-resource.openai.azure.com/
OPENAI_MODEL=your-deployment-name
```

**本地模型（Ollama、LM Studio）**：
```env
OPENAI_API_KEY=not-needed
OPENAI_BASE_URL=http://localhost:11434/v1
OPENAI_MODEL=llama2
```

## API端点

### 会话管理

- `POST /api/sessions/new` - 创建新会话
- `GET /api/sessions` - 列出所有会话
- `DELETE /api/sessions/:id` - 删除会话
- `GET /api/sessions/:id/history` - 获取会话消息

### 聊天

- `POST /api/chat` - 发送消息
  ```json
  {
    "session_id": "uuid",
    "message": "your message"
  }
  ```
  响应：
  ```json
  {
    "response": "AI response text"
  }
  ```

## 功能说明

### 会话管理

每个会话是独立的，拥有：
- 唯一的UUID标识符
- 消息历史记录
- ChatAgent实例
- 持久化存储（JSON文件）

会话在每条消息后自动保存，并在服务器重启时加载。

### 聊天Agent

应用使用自定义的`SimpleChatAgent`：
- 自动维护对话上下文
- 直接LLM集成，确保可靠性
- 系统消息支持
- 线程安全的会话历史
- 支持任何OpenAI兼容API

### 本地存储

所有会话存储为JSON文件在`sessions/`目录：
```
sessions/
├── 123e4567-e89b-12d3-a456-426614174000.json
├── 234e5678-f90c-23e4-b567-537725285111.json
└── ...
```

每个文件包含：
- 会话元数据（ID、时间戳）
- 完整消息历史
- 启动时自动加载

## 自定义

### 更改LLM模型

使用环境变量：
```env
OPENAI_MODEL=gpt-4o
```

或编辑`main.go:99-102`。

### 更改系统提示词

编辑`main.go:27-31`的`NewSimpleChatAgent`函数：
```go
systemMsg := llms.MessageContent{
    Role:  llms.ChatMessageTypeSystem,
    Parts: []llms.ContentPart{llms.TextPart("你的自定义系统消息")},
}
```

### 使用不同的LLM提供商

在`.env`中设置base URL：
```env
OPENAI_BASE_URL=https://your-provider.com/v1
OPENAI_API_KEY=your-provider-key
OPENAI_MODEL=your-model-name
```

## 故障排除

### "OPENAI_API_KEY environment variable not set"

确保你已经创建了`.env`文件并添加了API密钥：
```bash
cp .env.example .env
# 编辑.env并添加你的密钥
```

### 端口已被占用

在`.env`中更改端口：
```env
PORT=3000
```

### 会话未加载

检查`sessions/`目录是否存在且有正确的权限：
```bash
ls -la sessions/
```

## 最近更新

查看[CHANGELOG.md](CHANGELOG.md)了解详细更改。

### 最新更新（2025-12-09）
- ✅ **自动创建首个会话**：没有会话时自动创建并选中一个会话
- ✅ 使用SimpleChatAgent修复了LLM集成问题
- ✅ 添加了对OpenAI兼容API的支持（百度、Azure等）
- ✅ 移除了清空历史功能（简化为仅删除）
- ✅ 改进了错误处理和日志记录
- ✅ 添加了OPENAI_BASE_URL和OPENAI_MODEL配置

## 待办事项

- [ ] 在UI中添加流式响应支持
- [ ] 会话导出/导入功能
- [ ] 跨会话搜索
- [ ] 消息的Markdown渲染
- [ ] 代码语法高亮
- [ ] 语音输入/输出
- [ ] 会话标签和组织
- [ ] 多用户支持和身份验证
- [ ] 工具/函数调用支持

## 许可证

本项目是LangGraphGo的一部分，遵循相同的许可证。

## 了解更多

- [LangGraphGo文档](https://github.com/smallnest/langgraphgo)
- [LangChain Go](https://github.com/tmc/langchaingo)
