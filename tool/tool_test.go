package tool

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTavilySearch_Interface(t *testing.T) {
	os.Setenv("TAVILY_API_KEY", "test-key")
	defer os.Unsetenv("TAVILY_API_KEY")

	tool, err := NewTavilySearch("")
	require.NoError(t, err)
	assert.Equal(t, "Tavily_Search", tool.Name())
	assert.NotEmpty(t, tool.Description())
}

func TestExaSearch_Interface(t *testing.T) {
	os.Setenv("EXA_API_KEY", "test-key")
	defer os.Unsetenv("EXA_API_KEY")

	tool, err := NewExaSearch("")
	require.NoError(t, err)
	assert.Equal(t, "Exa_Search", tool.Name())
	assert.NotEmpty(t, tool.Description())
}

// Helper to mock Tavily API
func mockTavilyServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"results": [
				{
					"title": "Test Result",
					"url": "http://example.com",
					"content": "This is a test content."
				}
			]
		}`))
	}))
}

func TestTavilySearch_Call(t *testing.T) {
	server := mockTavilyServer()
	defer server.Close()

	os.Setenv("TAVILY_API_KEY", "test-key")
	defer os.Unsetenv("TAVILY_API_KEY")

	tool, err := NewTavilySearch("", WithTavilyBaseURL(server.URL))
	require.NoError(t, err)

	result, err := tool.Call(context.Background(), "test query")
	require.NoError(t, err)
	assert.Contains(t, result, "Title: Test Result")
	assert.Contains(t, result, "URL: http://example.com")
	assert.Contains(t, result, "Content: This is a test content.")
}

// Helper to mock Exa API
func mockExaServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"results": [
				{
					"title": "Exa Result",
					"url": "http://exa.example.com",
					"text": "This is exa content."
				}
			]
		}`))
	}))
}

func TestExaSearch_Call(t *testing.T) {
	server := mockExaServer()
	defer server.Close()

	os.Setenv("EXA_API_KEY", "test-key")
	defer os.Unsetenv("EXA_API_KEY")

	tool, err := NewExaSearch("", WithExaBaseURL(server.URL))
	require.NoError(t, err)

	result, err := tool.Call(context.Background(), "test query")
	require.NoError(t, err)
	assert.Contains(t, result, "Title: Exa Result")
	assert.Contains(t, result, "URL: http://exa.example.com")
	assert.Contains(t, result, "Content: This is exa content.")
}

func TestBochaSearch_Interface(t *testing.T) {
	os.Setenv("BOCHA_API_KEY", "test-key")
	defer os.Unsetenv("BOCHA_API_KEY")

	tool, err := NewBochaSearch("")
	require.NoError(t, err)
	assert.Equal(t, "Bocha_Search", tool.Name())
	assert.NotEmpty(t, tool.Description())
}

// Helper to mock Bocha API
func mockBochaServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/web-search" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": {
				"webPages": {
					"value": [
						{
							"name": "Bocha Result",
							"url": "http://bocha.example.com",
							"snippet": "This is bocha content."
						}
					]
				}
			}
		}`))
	}))
}

func TestBochaSearch_Call(t *testing.T) {
	server := mockBochaServer()
	defer server.Close()

	os.Setenv("BOCHA_API_KEY", "test-key")
	defer os.Unsetenv("BOCHA_API_KEY")

	tool, err := NewBochaSearch("", WithBochaBaseURL(server.URL+"/v1/web-search"))
	require.NoError(t, err)

	result, err := tool.Call(context.Background(), "test query")
	require.NoError(t, err)
	assert.Contains(t, result, "Title: Bocha Result")
	assert.Contains(t, result, "URL: http://bocha.example.com")
	assert.Contains(t, result, "Content: This is bocha content.")
}
