package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitHubOAuthAuthorizeURL(t *testing.T) {
	oauth := GitHubOAuth{ClientID: "client-1", ClientSecret: "secret-1", RedirectURI: "http://localhost:8080/auth/github/callback"}
	got := oauth.AuthorizeURL("state-abc")

	for _, want := range []string{
		"https://github.com/login/oauth/authorize?",
		"client_id=client-1",
		"redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fauth%2Fgithub%2Fcallback",
		"scope=user%3Aemail",
		"state=state-abc",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("authorize url missing %q: %s", want, got)
		}
	}
}

func TestGitHubOAuthExchangeCode(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/token" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "gh-token-1"})
		}))
		defer server.Close()

		oauth := GitHubOAuth{TokenURL: server.URL + "/token", HTTPClient: server.Client()}
		token, err := oauth.ExchangeCode(context.Background(), "code-1")
		if err != nil {
			t.Fatalf("exchange code: %v", err)
		}
		if token != "gh-token-1" {
			t.Fatalf("unexpected token: %q", token)
		}
	})

	t.Run("http error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "bad_verification_code", http.StatusBadRequest)
		}))
		defer server.Close()

		oauth := GitHubOAuth{TokenURL: server.URL, HTTPClient: server.Client()}
		if _, err := oauth.ExchangeCode(context.Background(), "bad-code"); err == nil {
			t.Fatal("expected error for rejected code")
		}
	})
}

func TestGitHubOAuthFetchUser(t *testing.T) {
	t.Run("uses public email", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/user" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			email := "alice@example.com"
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 12345, "login": "alice", "email": email})
		}))
		defer server.Close()

		oauth := GitHubOAuth{UserURL: server.URL + "/user", HTTPClient: server.Client()}
		user, err := oauth.FetchUser(context.Background(), "gh-token-1")
		if err != nil {
			t.Fatalf("fetch user: %v", err)
		}
		if user.ID != "12345" || user.Login != "alice" || user.Email != "alice@example.com" {
			t.Fatalf("unexpected user: %+v", user)
		}
	})

	t.Run("falls back to primary verified email", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 12345, "login": "alice", "email": nil})
		})
		mux.HandleFunc("/emails", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"email": "alice@example.com", "primary": true, "verified": true},
			})
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		oauth := GitHubOAuth{UserURL: server.URL + "/user", EmailsURL: server.URL + "/emails", HTTPClient: server.Client()}
		user, err := oauth.FetchUser(context.Background(), "gh-token-1")
		if err != nil {
			t.Fatalf("fetch user: %v", err)
		}
		if user.Email != "alice@example.com" {
			t.Fatalf("expected primary email fallback, got %q", user.Email)
		}
	})
}
