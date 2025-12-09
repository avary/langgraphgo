# LangGraphGo Chat Agent

A web-based multi-session chat application with persistent local history. Supports OpenAI and OpenAI-compatible APIs (Baidu, Azure, local models, etc.).

## Features

- 🔄 **Multi-Session Support**: Create and manage multiple independent chat sessions
- 💾 **Persistent History**: All conversations are automatically saved to local disk
- 🌐 **Web Interface**: Clean, modern web UI for easy interaction
- 🤖 **Smart Chat Agent**: Lightweight chat agent with conversation history management
- 🔌 **Multi-Provider Support**: Works with OpenAI, Baidu, Azure, and any OpenAI-compatible API
- 🎨 **Beautiful UI**: Responsive design with smooth animations
- 📝 **Session Management**: Create, view, clear, and delete sessions
- ⏱️ **Real-time Updates**: See message counts and timestamps

## Architecture

```
showcases/chat/
├── main.go              # HTTP server and ChatAgent integration
├── session.go           # Session management and persistence
├── static/
│   └── index.html      # Web frontend
├── sessions/           # Local session storage (auto-created)
├── go.mod
├── .env                # Configuration (create from .env.example)
└── README.md
```

## Quick Start

### 1. Setup

```bash
cd showcases/chat

# Copy environment template
cp .env.example .env

# Edit .env and add your OpenAI API key
# OPENAI_API_KEY=sk-...
```

### 2. Install Dependencies

```bash
go mod tidy
```

### 3. Run the Server

```bash
go run main.go session.go
```

The server will start at `http://localhost:8080`

### 4. Use the Web Interface

1. Open your browser to `http://localhost:8080`
2. A session will be automatically created if none exists
3. Start chatting immediately!

## Configuration

Environment variables (in `.env`):

```env
# Required: Your API key
OPENAI_API_KEY=your-api-key-here

# Optional: Model name (default: gpt-4o-mini)
OPENAI_MODEL=gpt-4o-mini

# Optional: Base URL for OpenAI-compatible APIs
# Leave empty for standard OpenAI API
# Examples:
#   Baidu: https://your-baidu-endpoint/v1
#   Azure: https://your-resource.openai.azure.com/
#   Ollama: http://localhost:11434/v1
OPENAI_BASE_URL=

# Optional: Server port (default: 8080)
PORT=8080

# Optional: Session storage directory (default: ./sessions)
SESSION_DIR=./sessions

# Optional: Maximum messages per session (default: 50)
MAX_HISTORY_SIZE=50
```

### Using Different LLM Providers

**OpenAI (default)**:
```env
OPENAI_API_KEY=sk-your-openai-key
```

**Baidu Qianfan**:
```env
OPENAI_API_KEY=your-baidu-token
OPENAI_BASE_URL=https://your-baidu-endpoint/v1
OPENAI_MODEL=your-model-name
```

**Azure OpenAI**:
```env
OPENAI_API_KEY=your-azure-key
OPENAI_BASE_URL=https://your-resource.openai.azure.com/
OPENAI_MODEL=your-deployment-name
```

**Local Models (Ollama, LM Studio)**:
```env
OPENAI_API_KEY=not-needed
OPENAI_BASE_URL=http://localhost:11434/v1
OPENAI_MODEL=llama2
```

## API Endpoints

### Sessions

- `POST /api/sessions/new` - Create a new session
- `GET /api/sessions` - List all sessions
- `DELETE /api/sessions/:id` - Delete a session
- `GET /api/sessions/:id/history` - Get session messages

### Chat

- `POST /api/chat` - Send a message
  ```json
  {
    "session_id": "uuid",
    "message": "your message"
  }
  ```
  Response:
  ```json
  {
    "response": "AI response text"
  }
  ```

## Features Explained

### Session Management

Each session is independent with its own:
- Unique UUID identifier
- Message history
- ChatAgent instance
- Persistent storage (JSON file)

Sessions are automatically saved after each message and loaded on server restart.

### Chat Agent

The application uses a custom `SimpleChatAgent`:
- Maintains conversation context automatically
- Direct LLM integration for reliability
- System message support
- Thread-safe conversation history
- Works with any OpenAI-compatible API

### Local Storage

All sessions are stored as JSON files in the `sessions/` directory:
```
sessions/
├── 123e4567-e89b-12d3-a456-426614174000.json
├── 234e5678-f90c-23e4-b567-537725285111.json
└── ...
```

Each file contains:
- Session metadata (ID, timestamps)
- Full message history
- Automatically loaded on startup

## Customization

### Change the LLM Model

Use environment variable:
```env
OPENAI_MODEL=gpt-4o
```

Or edit `main.go:99-102`.

### Change System Prompt

Edit `main.go:27-31` in the `NewSimpleChatAgent` function:
```go
systemMsg := llms.MessageContent{
    Role:  llms.ChatMessageTypeSystem,
    Parts: []llms.ContentPart{llms.TextPart("Your custom system message here")},
}
```

### Use Different LLM Provider

Set the base URL in `.env`:
```env
OPENAI_BASE_URL=https://your-provider.com/v1
OPENAI_API_KEY=your-provider-key
OPENAI_MODEL=your-model-name
```

## Development

### Project Structure

- **main.go**: HTTP server, routing, ChatAgent management
- **session.go**: Session persistence and history management
- **static/index.html**: Single-page web application
- **sessions/**: Auto-created directory for session storage

### Testing

```bash
# Run the server
go run main.go session.go

# In another terminal, test the API
curl -X POST http://localhost:8080/api/sessions/new

curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id":"...", "message":"Hello!"}'
```

## Troubleshooting

### "OPENAI_API_KEY environment variable not set"

Make sure you've created `.env` file and added your API key:
```bash
cp .env.example .env
# Edit .env and add your key
```

### Port already in use

Change the port in `.env`:
```env
PORT=3000
```

### Sessions not loading

Check that the `sessions/` directory exists and has proper permissions:
```bash
ls -la sessions/
```

## Recent Changes

See [CHANGELOG.md](CHANGELOG.md) for detailed changes.

### Latest Updates (2025-12-09)
- ✅ **Auto-create first session**: Automatically creates and selects a session when none exists
- ✅ Fixed LLM integration with simplified SimpleChatAgent
- ✅ Added support for OpenAI-compatible APIs (Baidu, Azure, etc.)
- ✅ Removed clear history feature (simplified to delete only)
- ✅ Improved error handling and logging
- ✅ Added OPENAI_BASE_URL and OPENAI_MODEL configuration

## TODO / Future Enhancements

- [ ] Add streaming support to UI
- [ ] Export/import session functionality
- [ ] Search across all sessions
- [ ] Markdown rendering in messages
- [ ] Code syntax highlighting
- [ ] Voice input/output
- [ ] Session tagging and organization
- [ ] Multi-user support with authentication
- [ ] Tool/function calling support

## License

This project is part of LangGraphGo and follows the same license.

## Learn More

- [LangGraphGo Documentation](https://github.com/smallnest/langgraphgo)
- [ChatAgent API Reference](../../prebuilt/chat_agent.go)
- [LangChain Go](https://github.com/tmc/langchaingo)
