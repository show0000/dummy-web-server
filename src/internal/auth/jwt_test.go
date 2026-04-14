package auth

import (
	"testing"
	"time"
)

func newTestService() *JWTService {
	return NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour)
}

func TestGenerateTokenPair(t *testing.T) {
	svc := newTestService()
	pair, err := svc.GenerateTokenPair("alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if pair.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestValidateAccessToken(t *testing.T) {
	svc := newTestService()
	pair, _ := svc.GenerateTokenPair("bob")

	username, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "bob" {
		t.Errorf("expected bob, got %s", username)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	svc := newTestService()
	_, err := svc.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidateExpiredToken(t *testing.T) {
	svc := NewJWTService("test-secret", 1*time.Millisecond, 7*24*time.Hour)
	pair, _ := svc.GenerateTokenPair("alice")

	time.Sleep(10 * time.Millisecond)

	_, err := svc.ValidateAccessToken(pair.AccessToken)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	svc := newTestService()
	pair1, _ := svc.GenerateTokenPair("carol")

	// Refresh should work with valid refresh token
	pair2, err := svc.Refresh(pair1.RefreshToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair2.AccessToken == "" || pair2.RefreshToken == "" {
		t.Error("expected non-empty tokens after refresh")
	}

	// Old refresh token should be revoked (rotation)
	_, err = svc.Refresh(pair1.RefreshToken)
	if err == nil {
		t.Fatal("expected error: old refresh token should be revoked")
	}
}

func TestRefreshInvalidToken(t *testing.T) {
	svc := newTestService()
	_, err := svc.Refresh("not-a-valid-refresh-token")
	if err == nil {
		t.Fatal("expected error for invalid refresh token")
	}
}

func TestLogoutBlacklistsAccessToken(t *testing.T) {
	svc := newTestService()
	pair, _ := svc.GenerateTokenPair("dave")

	// Before logout, access token is valid
	_, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("expected valid before logout: %v", err)
	}

	// Logout
	svc.Logout(pair.AccessToken, pair.RefreshToken)

	// After logout, access token is blacklisted
	_, err = svc.ValidateAccessToken(pair.AccessToken)
	if err == nil {
		t.Fatal("expected error: access token should be blacklisted after logout")
	}

	// Refresh token should also be revoked
	_, err = svc.Refresh(pair.RefreshToken)
	if err == nil {
		t.Fatal("expected error: refresh token should be revoked after logout")
	}
}

func TestDifferentSecretsReject(t *testing.T) {
	svc1 := NewJWTService("secret-a", 15*time.Minute, 7*24*time.Hour)
	svc2 := NewJWTService("secret-b", 15*time.Minute, 7*24*time.Hour)

	pair, _ := svc1.GenerateTokenPair("eve")
	_, err := svc2.ValidateAccessToken(pair.AccessToken)
	if err == nil {
		t.Fatal("expected error: token signed with different secret should be rejected")
	}
}
