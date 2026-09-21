package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/futureagi/agentcc-gateway/internal/config"
	"github.com/futureagi/agentcc-gateway/internal/pipeline"
	"github.com/futureagi/agentcc-gateway/internal/providers"
)

func TestAnthropicMessagesRoutesClaudeAliasToGemini(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		if r.URL.Path != "/v1beta/models/gemini-3.7-flash:generateContent" {
			t.Errorf("upstream path = %q", r.URL.Path)
			http.Error(w, "wrong model", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"GEMINI_OK"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":2,"totalTokenCount":7}}`))
	}))
	defer upstream.Close()

	cfg := config.DefaultConfig()
	cfg.Auth.Enabled = true
	cfg.Auth.Keys = []config.AuthKeyConfig{{
		Name:    "alk-test",
		Key:     "test-virtual-key",
		Owner:   "test-org",
		KeyType: "internal",
		Metadata: map[string]string{
			"access_groups": "alk_authoring",
		},
	}}
	cfg.Routing.AccessGroups = config.AccessGroupsConfig{
		"alk_authoring": {
			Models:  []string{"vertex_ai/gemini-3.7-flash"},
			Aliases: map[string]string{"claude-sonnet-4-6": "vertex_ai/gemini-3.7-flash"},
		},
	}
	cfg.Providers["vertex"] = config.ProviderConfig{
		BaseURL:   upstream.URL,
		APIKey:    "test-key",
		APIFormat: "gemini",
		Models:    []string{"vertex_ai/gemini-3.7-flash"},
	}
	registry, err := providers.NewRegistry(cfg)
	if err != nil {
		t.Fatalf("creating registry: %v", err)
	}
	srv := New(cfg, "", registry, pipeline.NewEngine(), nil, nil, nil, nil, testModelDBPtr(), nil, nil)
	srv.ready.Store(true)

	request := httptest.NewRequest("POST", "/v1/messages", bytes.NewBufferString(
		`{"model":"claude-sonnet-4-6","max_tokens":32,"messages":[{"role":"user","content":"Say hi"}]}`,
	))
	request.Header.Set("Authorization", "Bearer test-virtual-key")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("anthropic-version", "2023-06-01")
	response := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "GEMINI_OK") || !strings.Contains(response.Body.String(), `"model":"claude-sonnet-4-6"`) {
		t.Fatalf("unexpected Anthropic response: %s", response.Body.String())
	}
}
