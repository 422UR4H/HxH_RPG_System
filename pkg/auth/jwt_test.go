package auth_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"github.com/google/uuid"
)

func TestGenerateToken(t *testing.T) {
	userID := uuid.New()

	token, err := auth.GenerateToken(userID)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken() returned empty token")
	}
}

func TestGenerateToken_DifferentUsersProduceDifferentTokens(t *testing.T) {
	token1, _ := auth.GenerateToken(uuid.New())
	token2, _ := auth.GenerateToken(uuid.New())

	if token1 == token2 {
		t.Error("expected different tokens for different users")
	}
}

func TestValidateToken_Success(t *testing.T) {
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
		t.Errorf("got UserID %v, want %v", claims.UserID, userID)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}

	expiresIn := time.Until(claims.ExpiresAt.Time)
	if expiresIn < 23*time.Hour || expiresIn > 25*time.Hour {
		t.Errorf("expected expiration ~24h from now, got %v", expiresIn)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	_, err := auth.ValidateToken("invalid.token.string")
	if err == nil {
		t.Error("expected error for invalid token, got nil")
	}
}

func TestValidateToken_EmptyToken(t *testing.T) {
	_, err := auth.ValidateToken("")
	if err == nil {
		t.Error("expected error for empty token, got nil")
	}
}

func TestValidateToken_TamperedToken(t *testing.T) {
	token, _ := auth.GenerateToken(uuid.New())

	// tamper with the last character
	tampered := token[:len(token)-1] + "X"

	_, err := auth.ValidateToken(tampered)
	if err == nil {
		t.Error("expected error for tampered token, got nil")
	}
}
