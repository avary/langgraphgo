// Package nativeopenai is an OpenAI-compatible LLM provider that implements
// github.com/smallnest/langgraphgo/llms.Model directly, with no dependency
// on github.com/tmc/langchaingo. It exists both as a first-class provider and
// as proof that the framework's Model interface stands on its own: a provider
// can be written against the llms package alone and plugged straight into the graph
// engine, prebuilt agents, and memory without any adapter.
//
// It uses github.com/sashabaranov/go-openai purely as an HTTP transport for the
// Chat Completions API, so it also works against OpenAI-compatible endpoints
// (Azure OpenAI, local servers, and other vendors) via WithBaseURL.
package nativeopenai

import (
	"errors"
	"os"

	goopenai "github.com/sashabaranov/go-openai"
	"github.com/smallnest/langgraphgo/llms"
)

// ErrMissingToken is returned by New when no API token is configured.
var ErrMissingToken = errors.New("nativeopenai: missing API token (set OPENAI_API_KEY or use WithToken)")

// DefaultModel is used when no model is set via WithModel or per-call WithModel.
const DefaultModel = "gpt-4o-mini"

// LLM is an OpenAI-compatible model implementing llms.Model.
type LLM struct {
	client *goopenai.Client
	model  string
}

// Compile-time proof that LLM satisfies the framework Model interface with no
// langchaingo involvement.
var _ llms.Model = (*LLM)(nil)

type config struct {
	token   string
	baseURL string
	orgID   string
	model   string
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

// New creates a new native OpenAI provider.
func New(opts ...Option) (*LLM, error) {
	cfg := config{
		token: os.Getenv("OPENAI_API_KEY"),
		model: DefaultModel,
	}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.token == "" {
		return nil, ErrMissingToken
	}

	clientCfg := goopenai.DefaultConfig(cfg.token)
	if cfg.baseURL != "" {
		clientCfg.BaseURL = cfg.baseURL
	}
	if cfg.orgID != "" {
		clientCfg.OrgID = cfg.orgID
	}

	return &LLM{
		client: goopenai.NewClientWithConfig(clientCfg),
		model:  cfg.model,
	}, nil
}
