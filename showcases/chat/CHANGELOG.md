# Changelog

All notable changes to the LangGraphGo Chat project will be documented in this file.

## [2025-12-09] - Auto-Create First Session

### Added
- **Auto-create first session**: When user opens the app with no sessions, automatically creates one
- **Auto-select session**: Automatically selects the first session when loading the page

### Improved
- Better user experience - no need to manually create first session
- Sessions are ready to use immediately upon opening the app

## [2025-12-09] - Fixed and Simplified

### Fixed
- **Fixed LLM integration issue**: Replaced `prebuilt.ChatAgent` with custom `SimpleChatAgent` that directly uses `llms.Model` for more reliable operation
- **Fixed type assertion error**: Corrected response handling to work with langchaingo's actual response types
- **Added comprehensive logging**: Added detailed logging for debugging chat requests and responses

### Changed
- **Simplified architecture**: Removed dependency on LangGraphGo's prebuilt ChatAgent to avoid graph/tool complexity
- **Removed clear history feature**: Simplified UI by keeping only the delete button for sessions (per user request)
- **Improved error handling**: Better error messages and logging throughout the application

### Added
- **Support for OpenAI-compatible APIs**: Added `OPENAI_BASE_URL` configuration to support Baidu, Azure, and other OpenAI-compatible providers
- **Custom model selection**: Added `OPENAI_MODEL` environment variable for model selection
- **SimpleChatAgent**: New lightweight chat agent implementation with conversation history management

### Technical Details
- Replaced graph-based agent with direct LLM calls
- Simplified message history management
- Removed unused streaming endpoint
- Added system message support
- Improved session management

## [2025-12-09] - Initial Release

### Added
- Multi-session chat support
- Persistent session storage (JSON files)
- Web-based UI with session management
- RESTful API endpoints
- Session CRUD operations
- Message history tracking
- Beautiful responsive design
- Real-time session updates

### Features
- Create/delete chat sessions
- Send messages and receive responses
- View message history
- Session metadata (message count, timestamps)
- Automatic session persistence
- Environment-based configuration
