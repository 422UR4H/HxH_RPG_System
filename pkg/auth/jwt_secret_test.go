package auth_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"github.com/google/uuid"
)

func TestJWTSecret_SameSecretValidates(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret-a")
	userID := uuid.New()

	token, err := auth.GenerateToken(userID)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
}

func TestJWTSecret_DifferentSecretFailsValidation(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret-a")
	token, err := auth.GenerateToken(uuid.New())
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	t.Setenv("JWT_SECRET", "secret-b")
	if claims, err := auth.ValidateToken(token); err == nil || claims != nil {
		t.Errorf("expected validation to fail under another secret, got claims=%v err=%v", claims, err)
	}
}

func TestJWTSecret_UnsetFallsBackToDevKey(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	userID := uuid.New()

	token, err := auth.GenerateToken(userID)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	claims, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}

	// A token signed with the dev key must not validate under a real secret.
	t.Setenv("JWT_SECRET", "real-secret")
	if _, err := auth.ValidateToken(token); err == nil {
		t.Error("dev-key token must not validate under a configured secret")
	}
}

func TestUsingDevSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	if !auth.UsingDevSecret() {
		t.Error("UsingDevSecret() = false with JWT_SECRET unset, want true")
	}
	t.Setenv("JWT_SECRET", "something")
	if auth.UsingDevSecret() {
		t.Error("UsingDevSecret() = true with JWT_SECRET set, want false")
	}
}
