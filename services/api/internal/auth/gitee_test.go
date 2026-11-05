package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGiteeOAuthAuthorizeURL(t *testing.T) {
	oauth := GiteeOAuth{ClientID: "client-1", ClientSecret: "secret-1", RedirectURI: "http://localhost:8080/auth/gitee/callback"}
	got := oauth.AuthorizeURL("state-abc")

	for _, want := range []string{
		"https://gitee.com/oauth/authorize?",
		"client_id=client-1",
		"redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fauth%2Fgitee%2Fcallback",
		"response_type=code",
		"scope=user_info+emails",
		"state=state-abc",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("gitee authorize url missing %q: %s", want, got)
		}
	}
}

func TestGiteeOAuthExchangeCode(t *testing.T) {
	var gotUserAgent string
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		_ = r.ParseForm()
		gotForm = r.PostForm
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "gitee-token-1"})
	}))
	defer server.Close()

	oauth := GiteeOAuth{TokenURL: server.URL, HTTPClient: server.Client()}
	token, err := oauth.ExchangeCode(context.Background(), "code-1")
	if err != nil {
		t.Fatalf("exchange code: %v", err)
	}
	if token != "gitee-token-1" {
		t.Fatalf("unexpected token: %q", token)
	}
	if gotUserAgent == "" {
		t.Fatal("expected User-Agent header on token request")
	}
	if gotForm.Get("grant_type") != "authorization_code" || gotForm.Get("code") != "code-1" {
		t.Fatalf("unexpected token form: %v", gotForm)
	}
}

func TestGiteeOAuthFetchUser(t *testing.T) {
	t.Run("numeric id with public email", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("access_token") != "gitee-token-1" {
				t.Errorf("expected access_token query param, got %q", r.URL.RawQuery)
			}
			email := "alice@example.com"
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 12345, "login": "alice", "email": email})
		}))
		defer server.Close()

		oauth := GiteeOAuth{UserURL: server.URL, HTTPClient: server.Client()}
		user, err := oauth.FetchUser(context.Background(), "gitee-token-1")
		if err != nil {
			t.Fatalf("fetch user: %v", err)
		}
		if user.ID != "12345" || user.Login != "alice" || user.Email != "alice@example.com" {
			t.Fatalf("unexpected user: %+v", user)
		}
	})

	t.Run("string id and hidden email falls back to emails", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "abc-123", "login": "alice", "email": "未公开邮箱"})
		})
		mux.HandleFunc("/emails", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"email": "alice@example.com", "state": "confirmed", "scope": []string{"primary"}},
			})
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		oauth := GiteeOAuth{UserURL: server.URL + "/user", EmailsURL: server.URL + "/emails", HTTPClient: server.Client()}
		user, err := oauth.FetchUser(context.Background(), "gitee-token-1")
		if err != nil {
			t.Fatalf("fetch user: %v", err)
		}
		if user.ID != "abc-123" || user.Email != "alice@example.com" {
			t.Fatalf("expected id and primary email fallback, got %+v", user)
		}
	})
}
