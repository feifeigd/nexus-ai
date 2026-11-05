package logic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"entgo.io/ent/dialect"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"
	"nexusai/services/api/internal/types"
)

func TestGithubCallbackLogicCreatesAndReusesUser(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "gh-token-1"})
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		email := "alice@example.com"
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 12345, "login": "alice", "email": email})
	})
	gh := httptest.NewServer(mux)
	defer gh.Close()

	client, closeData, err := data.NewData(&data.Data{Database: data.Database{
		Driver:       dialect.SQLite,
		Source:       "file:github_callback_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()

	cfg := &config.Config{
		Auth:       config.AuthConfig{AccessSecret: "test-secret-123"},
		WebBaseURL: "http://localhost:3000",
		GitHub: config.GitHubOAuthConfig{
			ClientID:     "client-1",
			ClientSecret: "secret-1",
			RedirectURI:  "http://localhost:8080/auth/github/callback",
			TokenURL:     gh.URL + "/token",
			UserURL:      gh.URL + "/user",
		},
	}
	svcCtx := svc.NewServiceContext(cfg, agent.MockRunner{}, nil, nil, nil, nil, data.NewUserStore(client))
	l := NewGithubCallbackLogic(context.Background(), svcCtx)

	resp, err := l.GithubCallback(&types.GithubCallbackInput{Code: "code-1"})
	if err != nil {
		t.Fatalf("github callback: %v", err)
	}
	if !strings.HasPrefix(resp.RedirectURL, "http://localhost:3000/auth/callback?") {
		t.Fatalf("unexpected redirect url: %s", resp.RedirectURL)
	}
	if !strings.Contains(resp.RedirectURL, "token=") ||
		!strings.Contains(resp.RedirectURL, "username=alice") ||
		!strings.Contains(resp.RedirectURL, "role=user") {
		t.Fatalf("redirect url missing auth params: %s", resp.RedirectURL)
	}

	// 第二次用同一 github_id 登录应复用已有用户，而不是再建一个。
	resp2, err := l.GithubCallback(&types.GithubCallbackInput{Code: "code-2"})
	if err != nil {
		t.Fatalf("github callback again: %v", err)
	}
	if !strings.Contains(resp2.RedirectURL, "username=alice") {
		t.Fatalf("second login should reuse the same user: %s", resp2.RedirectURL)
	}
}

func TestGithubCallbackLogicMissingCode(t *testing.T) {
	client, closeData, err := data.NewData(&data.Data{Database: data.Database{
		Driver:       dialect.SQLite,
		Source:       "file:github_callback_missing_test?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	defer closeData()

	svcCtx := svc.NewServiceContext(&config.Config{}, agent.MockRunner{}, nil, nil, nil, nil, data.NewUserStore(client))
	l := NewGithubCallbackLogic(context.Background(), svcCtx)

	resp, err := l.GithubCallback(&types.GithubCallbackInput{})
	if err != nil {
		t.Fatalf("github callback: %v", err)
	}
	if !strings.Contains(resp.RedirectURL, "error=") {
		t.Fatalf("expected error redirect, got %s", resp.RedirectURL)
	}
}
