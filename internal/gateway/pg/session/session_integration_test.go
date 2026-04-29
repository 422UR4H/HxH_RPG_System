//go:build integration

package session_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	pgSession "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/session"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
)

func TestCreateSession_HappyPath(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgSession.NewRepository(pool)
	ctx := context.Background()

	userUUID := pgtest.InsertTestUser(t, pool, "gon", "gon@hunter.com", "secret")
	uid, err := uuid.Parse(userUUID)
	if err != nil {
		t.Fatalf("uuid.Parse() unexpected error: %v", err)
	}

	if err := repo.CreateSession(ctx, uid, "token-abc-123"); err != nil {
		t.Fatalf("CreateSession() unexpected error: %v", err)
	}

	token, err := repo.GetSessionTokenByUserUUID(ctx, uid)
	if err != nil {
		t.Fatalf("GetSessionTokenByUserUUID() after create: %v", err)
	}
	if token != "token-abc-123" {
		t.Errorf("token = %q, want %q", token, "token-abc-123")
	}
}

func TestGetSessionTokenByUserUUID_Found(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgSession.NewRepository(pool)
	ctx := context.Background()

	userUUID := pgtest.InsertTestUser(t, pool, "killua", "killua@hunter.com", "secret")
	uid, err := uuid.Parse(userUUID)
	if err != nil {
		t.Fatalf("uuid.Parse() unexpected error: %v", err)
	}

	if err := repo.CreateSession(ctx, uid, "old-token"); err != nil {
		t.Fatalf("CreateSession(old) unexpected error: %v", err)
	}
	if err := repo.CreateSession(ctx, uid, "newest-token"); err != nil {
		t.Fatalf("CreateSession(newest) unexpected error: %v", err)
	}

	token, err := repo.GetSessionTokenByUserUUID(ctx, uid)
	if err != nil {
		t.Fatalf("GetSessionTokenByUserUUID() unexpected error: %v", err)
	}
	if token != "newest-token" {
		t.Errorf("token = %q, want %q (most recent)", token, "newest-token")
	}
}

func TestGetSessionTokenByUserUUID_NotFound(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgSession.NewRepository(pool)
	ctx := context.Background()

	_, err := repo.GetSessionTokenByUserUUID(ctx, uuid.New())
	if err == nil {
		t.Fatal("GetSessionTokenByUserUUID() expected error, got nil")
	}
	if !errors.Is(err, pgSession.ErrSessionNotFound) {
		t.Errorf("error = %v, want %v", err, pgSession.ErrSessionNotFound)
	}
}

func TestValidateSession_Valid(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgSession.NewRepository(pool)
	ctx := context.Background()

	userUUID := pgtest.InsertTestUser(t, pool, "kurapika", "kurapika@hunter.com", "secret")
	uid, err := uuid.Parse(userUUID)
	if err != nil {
		t.Fatalf("uuid.Parse() unexpected error: %v", err)
	}

	if err := repo.CreateSession(ctx, uid, "valid-token"); err != nil {
		t.Fatalf("CreateSession() unexpected error: %v", err)
	}

	valid, err := repo.ValidateSession(ctx, uid, "valid-token")
	if err != nil {
		t.Fatalf("ValidateSession() unexpected error: %v", err)
	}
	if !valid {
		t.Error("ValidateSession() = false, want true")
	}
}

func TestValidateSession_InvalidToken(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgSession.NewRepository(pool)
	ctx := context.Background()

	userUUID := pgtest.InsertTestUser(t, pool, "leorio", "leorio@hunter.com", "secret")
	uid, err := uuid.Parse(userUUID)
	if err != nil {
		t.Fatalf("uuid.Parse() unexpected error: %v", err)
	}

	if err := repo.CreateSession(ctx, uid, "real-token"); err != nil {
		t.Fatalf("CreateSession() unexpected error: %v", err)
	}

	valid, err := repo.ValidateSession(ctx, uid, "wrong-token")
	if err != nil {
		t.Fatalf("ValidateSession() unexpected error: %v", err)
	}
	if valid {
		t.Error("ValidateSession() = true, want false")
	}
}

func TestValidateSession_NoSessionForUser(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgSession.NewRepository(pool)
	ctx := context.Background()

	valid, err := repo.ValidateSession(ctx, uuid.New(), "any-token")
	if err != nil {
		t.Fatalf("ValidateSession() unexpected error: %v", err)
	}
	if valid {
		t.Error("ValidateSession() = true, want false")
	}
}
