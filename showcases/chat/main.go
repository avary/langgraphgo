package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// SimpleChatAgent manages conversation history for a session
type SimpleChatAgent struct {
	llm      llms.Model
	messages []llms.MessageContent
	mu       sync.RWMutex
}

// NewSimpleChatAgent creates a simple chat agent
func NewSimpleChatAgent(llm llms.Model) *SimpleChatAgent {
	// Add system message
	systemMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeSystem,
		Parts: []llms.ContentPart{llms.TextPart("You are a helpful AI assistant. Be concise and friendly.")},
	}

	return &SimpleChatAgent{
		llm:      llm,
		messages: []llms.MessageContent{systemMsg},
	}
}

// Chat sends a message and returns response
func (a *SimpleChatAgent) Chat(ctx context.Context, message string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Add user message
	userMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{llms.TextPart(message)},
	}
	a.messages = append(a.messages, userMsg)

	// Call LLM with full history
	response, err := a.llm.GenerateContent(ctx, a.messages)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	// Extract response text
	var responseText string
	if response != nil && len(response.Choices) > 0 {
		responseText = response.Choices[0].Content
	}

	if responseText == "" {
		return "", fmt.Errorf("empty response from LLM")
	}

	// Add assistant response to history
	assistantMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart(responseText)},
	}
	a.messages = append(a.messages, assistantMsg)

	return responseText, nil
}

// getClientID generates a unique client ID based on IP and User-Agent
func getClientID(r *http.Request) string {
	// Get client IP
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = strings.Split(r.RemoteAddr, ":")[0]
	}

	// Get User-Agent
	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		userAgent = "unknown"
	}

	// Create unique hash from IP + User-Agent
	h := md5.New()
	h.Write([]byte(clientIP + userAgent + "chat-salt"))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// ChatServer manages HTTP endpoints and chat agents
type ChatServer struct {
	sessionManager  *SessionManager
	agents          map[string]*SimpleChatAgent
	llm             llms.Model
	agentMu         sync.RWMutex
	port            string
	sessionManagers map[string]*SessionManager  // clientID -> SessionManager
	smMu            sync.RWMutex
}

// NewChatServer creates a new chat server
func NewChatServer(sessionDir string, maxHistory int, port string) (*ChatServer, error) {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// Get model name from environment or use default
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	// Get base URL from environment (for OpenAI-compatible APIs)
	baseURL := os.Getenv("OPENAI_BASE_URL")

	// Create OpenAI LLM (works with OpenAI-compatible APIs like Baidu)
	var llm llms.Model
	var err error

	if baseURL != "" {
		llm, err = openai.New(
			openai.WithModel(model),
			openai.WithToken(apiKey),
			openai.WithBaseURL(baseURL),
		)
	} else {
		llm, err = openai.New(
			openai.WithModel(model),
			openai.WithToken(apiKey),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	return &ChatServer{
		sessionManager:  NewSessionManager(sessionDir, maxHistory),
		agents:          make(map[string]*SimpleChatAgent),
		llm:             llm,
		port:            port,
		sessionManagers: make(map[string]*SessionManager),
	}, nil
}

// getSessionManager gets or creates a SessionManager for a specific client
func (cs *ChatServer) getSessionManager(clientID string) *SessionManager {
	cs.smMu.Lock()
	defer cs.smMu.Unlock()

	sm, exists := cs.sessionManagers[clientID]
	if !exists {
		clientSessionDir := fmt.Sprintf("%s/clients/%s", cs.sessionManager.sessionDir, clientID)
		sm = NewSessionManager(clientSessionDir, cs.sessionManager.maxHistory)
		cs.sessionManagers[clientID] = sm
	}
	return sm
}

// getOrCreateAgent gets an existing agent or creates a new one for a session
func (cs *ChatServer) getOrCreateAgent(sessionID string) (*SimpleChatAgent, error) {
	cs.agentMu.RLock()
	agent, exists := cs.agents[sessionID]
	cs.agentMu.RUnlock()

	if exists {
		return agent, nil
	}

	// Create new agent
	cs.agentMu.Lock()
	defer cs.agentMu.Unlock()

	// Double-check after acquiring write lock
	if agent, exists := cs.agents[sessionID]; exists {
		return agent, nil
	}

	// Create simple chat agent
	agent = NewSimpleChatAgent(cs.llm)
	cs.agents[sessionID] = agent
	return agent, nil
}

// handleIndex serves the main HTML page
func (cs *ChatServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, "static/index.html")
}

// handleNewSession creates a new chat session
func (cs *ChatServer) handleNewSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)
	sm := cs.getSessionManager(clientID)
	session := sm.CreateSession()

	// Set client ID cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "client_id",
		Value:    clientID,
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"session_id": session.ID,
		"client_id":  clientID,
	})
}

// handleListSessions returns all active sessions for the client
func (cs *ChatServer) handleListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)
	sm := cs.getSessionManager(clientID)
	sessions := sm.ListSessions()

	type SessionInfo struct {
		ID           string    `json:"id"`
		Title        string    `json:"title"`
		MessageCount int       `json:"message_count"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
	}

	sessionInfos := make([]SessionInfo, 0, len(sessions))
	for _, session := range sessions {
		// Get the first user message as title
		title := "新会话"
		for _, msg := range session.Messages {
			if msg.Role == "user" {
				// Convert string to rune slice to properly handle UTF-8 characters
				runes := []rune(msg.Content)
				if len(runes) > 20 {
					title = string(runes[:20]) + "..."
				} else {
					title = msg.Content
				}
				break
			}
		}

		sessionInfos = append(sessionInfos, SessionInfo{
			ID:           session.ID,
			Title:        title,
			MessageCount: len(session.Messages),
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessionInfos)
}

// handleDeleteSession deletes a session
func (cs *ChatServer) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)
	sm := cs.getSessionManager(clientID)

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	// Delete agent
	cs.agentMu.Lock()
	delete(cs.agents, sessionID)
	cs.agentMu.Unlock()

	// Delete session
	err := sm.DeleteSession(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetHistory retrieves chat history for a session
func (cs *ChatServer) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)
	sm := cs.getSessionManager(clientID)

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	sessionID = strings.TrimSuffix(sessionID, "/history")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	messages, err := sm.GetMessages(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// handleChat handles chat message requests
func (cs *ChatServer) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" || req.Message == "" {
		http.Error(w, "session_id and message are required", http.StatusBadRequest)
		return
	}

	clientID := getClientID(r)
	sm := cs.getSessionManager(clientID)

	log.Printf("Chat request for session %s: %s", req.SessionID, req.Message)

	// Verify session exists
	_, err := sm.GetSession(req.SessionID)
	if err != nil {
		log.Printf("Session not found: %s", req.SessionID)
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get or create agent for this session
	agent, err := cs.getOrCreateAgent(req.SessionID)
	if err != nil {
		log.Printf("Failed to create agent: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create agent: %v", err), http.StatusInternalServerError)
		return
	}

	// Add user message to history
	sm.AddMessage(req.SessionID, "user", req.Message)

	// Get response from agent
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := agent.Chat(ctx, req.Message)
	if err != nil {
		log.Printf("Chat error for session %s: %v", req.SessionID, err)
		http.Error(w, fmt.Sprintf("Chat failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Chat response for session %s: %s", req.SessionID, response)

	// Add assistant response to history
	sm.AddMessage(req.SessionID, "assistant", response)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response": response,
	})
}

// handleGetClientID returns the client ID for the current user
func (cs *ChatServer) handleGetClientID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)

	// Set client ID cookie if not already set
	_, err := r.Cookie("client_id")
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "client_id",
			Value:    clientID,
			Path:     "/",
			MaxAge:   86400 * 30, // 30 days
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"client_id": clientID,
	})
}

// Start starts the HTTP server
func (cs *ChatServer) Start() error {
	http.HandleFunc("/", cs.handleIndex)
	http.HandleFunc("/api/client-id", cs.handleGetClientID)
	http.HandleFunc("/api/sessions/new", cs.handleNewSession)
	http.HandleFunc("/api/sessions", cs.handleListSessions)
	http.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/history") {
			cs.handleGetHistory(w, r)
		} else if r.Method == http.MethodDelete {
			cs.handleDeleteSession(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	http.HandleFunc("/api/chat", cs.handleChat)

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	addr := ":" + cs.port
	log.Printf("Chat server starting on http://localhost%s", addr)
	return http.ListenAndServe(addr, nil)
}

func main() {
	// Load configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sessionDir := os.Getenv("SESSION_DIR")
	if sessionDir == "" {
		sessionDir = "./sessions"
	}

	maxHistory := 50
	if maxHistoryStr := os.Getenv("MAX_HISTORY_SIZE"); maxHistoryStr != "" {
		fmt.Sscanf(maxHistoryStr, "%d", &maxHistory)
	}

	// Create and start server
	server, err := NewChatServer(sessionDir, maxHistory, port)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
