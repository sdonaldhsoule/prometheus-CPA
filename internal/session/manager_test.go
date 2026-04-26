package session

import "testing"

func TestManagerRoundTrip(t *testing.T) {
	t.Helper()

	manager := NewManager("test-secret", "donation_session")
	token, err := manager.Issue(&Claims{
		LocalUserID:  7,
		NewAPIUserID: "42",
		Username:     "alice",
		Email:        "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	claims, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if claims.LocalUserID != 7 {
		t.Fatalf("expected LocalUserID 7, got %d", claims.LocalUserID)
	}
	if claims.NewAPIUserID != "42" {
		t.Fatalf("expected NewAPIUserID 42, got %s", claims.NewAPIUserID)
	}
	if claims.Username != "alice" {
		t.Fatalf("expected username alice, got %s", claims.Username)
	}
	if claims.Email != "alice@example.com" {
		t.Fatalf("expected email alice@example.com, got %s", claims.Email)
	}
}

func TestManagerRejectsTamperedToken(t *testing.T) {
	t.Helper()

	manager := NewManager("test-secret", "donation_session")
	token, err := manager.Issue(&Claims{
		LocalUserID:  7,
		NewAPIUserID: "42",
		Username:     "alice",
		Email:        "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Issue returned error: %v", err)
	}

	if _, err := manager.Parse(token + "tampered"); err == nil {
		t.Fatal("expected Parse to reject tampered token")
	}
}
