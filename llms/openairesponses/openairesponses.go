// Package openairesponses is an OpenAI Responses API provider that implements
// github.com/smallnest/langgraphgo/llmtypes.Model directly, with no dependency
// on github.com/tmc/langchaingo and no dependency on a third-party OpenAI SDK.
//
// It targets the /v1/responses endpoint (OpenAI's newer, stateful generation
// API) rather than /v1/chat/completions, and speaks raw HTTP so it also works
// against OpenAI-compatible endpoints via WithBaseURL. Because it satisfies
// llmtypes.Model, it can be plugged straight into the graph engine, prebuilt
// agents, and memory without any adapter.
package openairesponses

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/smallnest/langgraphgo/llmtypes"
)

// ErrMissingToken is returned by New when no API token is configured.
var ErrMissingToken = errors.New("openairesponses: missing API token (set OPENAI_API_KEY or use WithToken)")

// DefaultModel is used when no model is set via WithModel or per-call WithModel.
const DefaultModel = "gpt-4o-mini"

// DefaultBaseURL is the OpenAI API root used when WithBaseURL is not given.
const DefaultBaseURL = "https://api.openai.com/v1"

// LLM is an OpenAI Responses API model implementing llmtypes.Model.
type LLM struct {
	httpClient *http.Client
	token      string
	baseURL    string
	orgID      string
	model      string
}

// Compile-time proof that LLM satisfies the framework Model interface with no
// langchaingo involvement.
var _ llmtypes.Model = (*LLM)(nil)

type config struct {
	token      string
	baseURL    string
	orgID      string
	model      string
	httpClient *http.Client
}

// Option configures the provider.
type Option func(*config)

// WithToken sets the API token. Defaults to the OPENAI_API_KEY env var.
func WithToken(token string) Option {
	return func(c *config) { c.token = token }
}

// WithBaseURL overrides the API base URL for OpenAI-compatible endpoints.
func WithBaseURL(baseURL string) Option {
	return func(c *config) { c.baseURL = baseURL }
}

// WithOrganization sets the OpenAI organization ID.
func WithOrganization(orgID string) Option {
	return func(c *config) { c.orgID = orgID }
}

// WithModel sets the default model name used when a call does not specify one.
func WithModel(model string) Option {
	return func(c *config) { c.model = model }
}

// WithHTTPClient overrides the HTTP client used for requests.
func WithHTTPClient(client *http.Client) Option {
	return func(c *config) { c.httpClient = client }
}

// New creates a new OpenAI Responses API provider.
func New(opts ...Option) (*LLM, error) {
	cfg := config{
		token:   os.Getenv("OPENAI_API_KEY"),
		baseURL: DefaultBaseURL,
		model:   DefaultModel,
	}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.token == "" {
		return nil, ErrMissingToken
	}
	if cfg.httpClient == nil {
		cfg.httpClient = &http.Client{Timeout: 5 * time.Minute}
	}

	return &LLM{
		httpClient: cfg.httpClient,
		token:      cfg.token,
		baseURL:    strings.TrimRight(cfg.baseURL, "/"),
		orgID:      cfg.orgID,
		model:      cfg.model,
	}, nil
}
