package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/donation-station/donation-station/internal/database"
	"github.com/donation-station/donation-station/internal/newapi"
	"github.com/donation-station/donation-station/internal/session"
	"github.com/gin-gonic/gin"
)

func TestUserAuthMiddlewareRejectsUnauthenticatedRequests(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	server := &Server{
		sessionManager: session.NewManager("test-secret", session.DefaultCookieName),
	}

	router := gin.New()
	router.GET("/protected", server.userAuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestUserAuthMiddlewareInjectsCurrentUser(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	manager := session.NewManager("test-secret", session.DefaultCookieName)
	token, err := manager.Issue(&session.Claims{
		LocalUserID:  9,
		NewAPIUserID: "101",
		Username:     "alice",
		Email:        "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	server := &Server{sessionManager: manager}
	router := gin.New()
	router.GET("/protected", server.userAuthMiddleware(), func(c *gin.Context) {
		user, ok := currentUserFromContext(c)
		if !ok {
			t.Fatal("expected current user in context")
		}
		c.JSON(http.StatusOK, gin.H{
			"id":       user.LocalUserID,
			"username": user.Username,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.DefaultCookieName,
		Value: token,
		Path:  "/",
	})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if body := resp.Body.String(); body != "{\"id\":9,\"username\":\"alice\"}" {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestUserLoginHandlerCreatesSessionCookie(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	authenticator := &fakeUserAuthenticator{
		user: &newapi.User{
			ID:          "42",
			Username:    "alice",
			Email:       "alice@example.com",
			DisplayName: "Alice",
		},
	}
	userStore := &fakeAppUserStore{
		user: &database.AppUser{
			ID:           9,
			NewAPIUserID: "42",
			Username:     "alice",
			Email:        "alice@example.com",
			DisplayName:  "Alice",
		},
	}

	server := &Server{
		sessionManager:      session.NewManager("test-secret", session.DefaultCookieName),
		userAuthenticator:   authenticator,
		appUserStore:        userStore,
		userCredentialStore: &fakeUserCredentialStore{},
	}

	router := gin.New()
	router.POST("/api/user/login", server.userLoginHandler)

	body := bytes.NewBufferString(`{"username":"alice","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", resp.Code, resp.Body.String())
	}
	if len(resp.Result().Cookies()) == 0 {
		t.Fatal("expected login response to set session cookie")
	}
	if authenticator.username != "alice" || authenticator.password != "secret" {
		t.Fatalf("unexpected authenticator input: %s / %s", authenticator.username, authenticator.password)
	}
	if userStore.lastUser == nil || userStore.lastUser.Username != "alice" {
		t.Fatal("expected app user store to upsert authenticated user")
	}
}

func TestUserCredentialHandlersUseCurrentSessionUser(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	manager := session.NewManager("test-secret", session.DefaultCookieName)
	token, err := manager.Issue(&session.Claims{
		LocalUserID:  9,
		NewAPIUserID: "42",
		Username:     "alice",
		Email:        "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	store := &fakeUserCredentialStore{
		credentials: []*database.Credential{
			{ID: 5, Email: "alice@example.com", Type: database.CredentialTypeCodex},
		},
		total: 1,
	}

	server := &Server{
		sessionManager:      manager,
		userCredentialStore: store,
	}

	router := gin.New()
	auth := router.Group("/api/user")
	auth.Use(server.userAuthMiddleware())
	auth.GET("/credentials", server.listMyCredentialsHandler)
	auth.DELETE("/credentials/:id", server.removeMyCredentialHandler)

	listReq := httptest.NewRequest(http.MethodGet, "/api/user/credentials?limit=10&offset=0", nil)
	listReq.AddCookie(&http.Cookie{Name: session.DefaultCookieName, Value: token})
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)

	if listResp.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d with body %s", listResp.Code, listResp.Body.String())
	}
	if store.listUserID != 9 {
		t.Fatalf("expected list to use current user 9, got %d", store.listUserID)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/user/credentials/5", nil)
	deleteReq.AddCookie(&http.Cookie{Name: session.DefaultCookieName, Value: token})
	deleteResp := httptest.NewRecorder()
	router.ServeHTTP(deleteResp, deleteReq)

	if deleteResp.Code != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d with body %s", deleteResp.Code, deleteResp.Body.String())
	}
	if store.removeUserID != 9 || store.removeCredentialID != 5 {
		t.Fatalf("expected delete to target user 9 credential 5, got user=%d credential=%d", store.removeUserID, store.removeCredentialID)
	}
}

type fakeUserAuthenticator struct {
	user     *newapi.User
	err      error
	username string
	password string
}

func (f *fakeUserAuthenticator) Authenticate(_ context.Context, username, password string) (*newapi.User, error) {
	f.username = username
	f.password = password
	return f.user, f.err
}

type fakeAppUserStore struct {
	user     *database.AppUser
	err      error
	lastUser *database.AppUser
}

func (f *fakeAppUserStore) UpsertAppUser(user *database.AppUser) (*database.AppUser, error) {
	f.lastUser = user
	return f.user, f.err
}

type fakeUserCredentialStore struct {
	credentials        []*database.Credential
	total              int
	err                error
	listUserID         int64
	removeUserID       int64
	removeCredentialID int64
}

func (f *fakeUserCredentialStore) ListUserCredentials(userID int64, limit, offset int) ([]*database.Credential, int, error) {
	f.listUserID = userID
	return f.credentials, f.total, f.err
}

func (f *fakeUserCredentialStore) RemoveUserCredential(userID, credentialID int64) (bool, error) {
	f.removeUserID = userID
	f.removeCredentialID = credentialID
	return true, f.err
}

func TestUserCredentialHandlersRejectInvalidCredentialID(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	manager := session.NewManager("test-secret", session.DefaultCookieName)
	token, err := manager.Issue(&session.Claims{
		LocalUserID:  9,
		NewAPIUserID: "42",
		Username:     "alice",
		Email:        "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	server := &Server{
		sessionManager:      manager,
		userCredentialStore: &fakeUserCredentialStore{},
	}

	router := gin.New()
	auth := router.Group("/api/user")
	auth.Use(server.userAuthMiddleware())
	auth.DELETE("/credentials/:id", server.removeMyCredentialHandler)

	req := httptest.NewRequest(http.MethodDelete, "/api/user/credentials/not-a-number", nil)
	req.AddCookie(&http.Cookie{Name: session.DefaultCookieName, Value: token})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestUserLoginHandlerRejectsMissingPassword(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	server := &Server{
		sessionManager:    session.NewManager("test-secret", session.DefaultCookieName),
		userAuthenticator: &fakeUserAuthenticator{},
		appUserStore:      &fakeAppUserStore{},
	}

	router := gin.New()
	router.POST("/api/user/login", server.userLoginHandler)

	body := bytes.NewBufferString(`{"username":"alice"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d with body %s", resp.Code, resp.Body.String())
	}
}

func TestUserMeHandlerReturnsSessionClaims(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	manager := session.NewManager("test-secret", session.DefaultCookieName)
	token, err := manager.Issue(&session.Claims{
		LocalUserID:  15,
		NewAPIUserID: "99",
		Username:     "bob",
		Email:        "bob@example.com",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	server := &Server{sessionManager: manager}
	router := gin.New()
	auth := router.Group("/api/user")
	auth.Use(server.userAuthMiddleware())
	auth.GET("/me", server.userMeHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	req.AddCookie(&http.Cookie{Name: session.DefaultCookieName, Value: token})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := int(payload["id"].(float64)); got != 15 {
		t.Fatalf("expected id 15, got %d", got)
	}
	if payload["username"] != "bob" {
		t.Fatalf("expected username bob, got %v", payload["username"])
	}
	if payload["newapi_user_id"] != "99" {
		t.Fatalf("expected newapi user id 99, got %v", payload["newapi_user_id"])
	}
}
