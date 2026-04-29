//go:build integration

package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	entityUser "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	pgUser "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/user"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/pgtest"
)

func TestCreateUser_HappyPath(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgUser.NewRepository(pool)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)
	u := &entityUser.User{
		UUID:      uuid.New(),
		Nick:      "gon",
		Email:     "gon@hunter.com",
		Password:  "secret123",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := repo.CreateUser(ctx, u); err != nil {
		t.Fatalf("CreateUser() unexpected error: %v", err)
	}

	got, err := repo.GetUserByEmail(ctx, u.Email)
	if err != nil {
		t.Fatalf("GetUserByEmail() after create: %v", err)
	}
	if got.Nick != "gon" {
		t.Errorf("nick = %q, want %q", got.Nick, "gon")
	}
	if got.Email != "gon@hunter.com" {
		t.Errorf("email = %q, want %q", got.Email, "gon@hunter.com")
	}
	if got.UUID != u.UUID {
		t.Errorf("uuid = %v, want %v", got.UUID, u.UUID)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgUser.NewRepository(pool)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)
	u1 := &entityUser.User{
		UUID:      uuid.New(),
		Nick:      "killua",
		Email:     "dup@hunter.com",
		Password:  "pass1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	u2 := &entityUser.User{
		UUID:      uuid.New(),
		Nick:      "kurapika",
		Email:     "dup@hunter.com",
		Password:  "pass2",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := repo.CreateUser(ctx, u1); err != nil {
		t.Fatalf("CreateUser(u1) unexpected error: %v", err)
	}

	err := repo.CreateUser(ctx, u2)
	if err == nil {
		t.Fatal("CreateUser(u2) expected error for duplicate email, got nil")
	}
}

func TestCreateUser_DuplicateNick(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgUser.NewRepository(pool)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)
	u1 := &entityUser.User{
		UUID:      uuid.New(),
		Nick:      "leorio",
		Email:     "leorio1@hunter.com",
		Password:  "pass1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	u2 := &entityUser.User{
		UUID:      uuid.New(),
		Nick:      "leorio",
		Email:     "leorio2@hunter.com",
		Password:  "pass2",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := repo.CreateUser(ctx, u1); err != nil {
		t.Fatalf("CreateUser(u1) unexpected error: %v", err)
	}

	err := repo.CreateUser(ctx, u2)
	if err == nil {
		t.Fatal("CreateUser(u2) expected error for duplicate nick, got nil")
	}
}

func TestGetUserByEmail_Found(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgUser.NewRepository(pool)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)
	rawPassword := "hunter2023"
	u := &entityUser.User{
		UUID:      uuid.New(),
		Nick:      "hisoka",
		Email:     "hisoka@hunter.com",
		Password:  rawPassword,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := repo.CreateUser(ctx, u); err != nil {
		t.Fatalf("CreateUser() unexpected error: %v", err)
	}

	got, err := repo.GetUserByEmail(ctx, "hisoka@hunter.com")
	if err != nil {
		t.Fatalf("GetUserByEmail() unexpected error: %v", err)
	}

	if got.Nick != "hisoka" {
		t.Errorf("nick = %q, want %q", got.Nick, "hisoka")
	}
	if got.Email != "hisoka@hunter.com" {
		t.Errorf("email = %q, want %q", got.Email, "hisoka@hunter.com")
	}
	if got.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if got.UUID != u.UUID {
		t.Errorf("uuid = %v, want %v", got.UUID, u.UUID)
	}

	// Password must be bcrypt-hashed, not the raw value
	if got.Password == rawPassword {
		t.Error("password stored as plaintext, expected bcrypt hash")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(got.Password), []byte(rawPassword)); err != nil {
		t.Errorf("bcrypt.CompareHashAndPassword() failed: %v", err)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgUser.NewRepository(pool)
	ctx := context.Background()

	_, err := repo.GetUserByEmail(ctx, "nonexistent@hunter.com")
	if err == nil {
		t.Fatal("GetUserByEmail() expected error, got nil")
	}
	if !errors.Is(err, pgUser.ErrEmailNotFound) {
		t.Errorf("error = %v, want %v", err, pgUser.ErrEmailNotFound)
	}
}

func TestExistsUserWithNick_True(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgUser.NewRepository(pool)
	ctx := context.Background()

	pgtest.InsertTestUser(t, pool, "bisky", "bisky@hunter.com", "pass")

	exists, err := repo.ExistsUserWithNick(ctx, "bisky")
	if err != nil {
		t.Fatalf("ExistsUserWithNick() unexpected error: %v", err)
	}
	if !exists {
		t.Error("ExistsUserWithNick() = false, want true")
	}
}

func TestExistsUserWithNick_False(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgUser.NewRepository(pool)
	ctx := context.Background()

	exists, err := repo.ExistsUserWithNick(ctx, "phantom")
	if err != nil {
		t.Fatalf("ExistsUserWithNick() unexpected error: %v", err)
	}
	if exists {
		t.Error("ExistsUserWithNick() = true, want false")
	}
}

func TestExistsUserWithEmail_True(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgUser.NewRepository(pool)
	ctx := context.Background()

	pgtest.InsertTestUser(t, pool, "kite", "kite@hunter.com", "pass")

	exists, err := repo.ExistsUserWithEmail(ctx, "kite@hunter.com")
	if err != nil {
		t.Fatalf("ExistsUserWithEmail() unexpected error: %v", err)
	}
	if !exists {
		t.Error("ExistsUserWithEmail() = false, want true")
	}
}
