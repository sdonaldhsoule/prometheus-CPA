package newapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientAuthenticate(t *testing.T) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/login":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST login, got %s", r.Method)
			}

			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode login payload: %v", err)
			}
			if payload["username"] != "alice" || payload["password"] != "secret" {
				t.Fatalf("unexpected login payload: %#v", payload)
			}

			http.SetCookie(w, &http.Cookie{
				Name:  "new-api-session",
				Value: "cookie-token",
				Path:  "/",
			})
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true}`))

		case "/api/user/self":
			cookie, err := r.Cookie("new-api-session")
			if err != nil {
				t.Fatalf("expected session cookie on self request: %v", err)
			}
			if cookie.Value != "cookie-token" {
				t.Fatalf("unexpected cookie value: %s", cookie.Value)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"42","username":"alice","email":"alice@example.com","display_name":"Alice"}}`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	user, err := client.Authenticate(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}

	if user.ID != "42" {
		t.Fatalf("expected user id 42, got %s", user.ID)
	}
	if user.Username != "alice" {
		t.Fatalf("expected username alice, got %s", user.Username)
	}
	if user.Email != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %s", user.Email)
	}
	if user.DisplayName != "Alice" {
		t.Fatalf("expected display name Alice, got %s", user.DisplayName)
	}
}

func TestClientAuthenticateReturnsErrorWhenLoginFails(t *testing.T) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/login" {
			http.NotFound(w, r)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid credentials"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)

	_, err := client.Authenticate(context.Background(), "alice", "bad-secret")
	if err == nil {
		t.Fatal("expected Authenticate to fail")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Fatalf("expected error to mention invalid credentials, got %v", err)
	}
}
