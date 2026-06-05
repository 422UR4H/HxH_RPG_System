package upload_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/upload"
	"github.com/google/uuid"
)

type mockR2Client struct {
	uploadURL   string
	publicURL   string
	err         error
	capturedKey string
}

func (m *mockR2Client) NewPresignedPutURL(_ context.Context, key string, _ time.Duration) (upload.PresignResult, error) {
	m.capturedKey = key
	return upload.PresignResult{UploadURL: m.uploadURL, PublicURL: m.publicURL}, m.err
}

func strPtr(s string) *string { return &s }

func TestPresignedURLHandler_InvalidFileType(t *testing.T) {
	mock := &mockR2Client{}
	handler := upload.PresignedURLHandler(mock)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	req := &upload.PresignedURLRequest{
		Body: upload.PresignedURLRequestBody{FileType: "invalid", SheetUUID: strPtr(uuid.New().String())},
	}

	_, err := handler(ctx, req)
	if err == nil {
		t.Error("expected error for invalid file_type")
	}
}

func TestPresignedURLHandler_ValidRequest(t *testing.T) {
	mock := &mockR2Client{uploadURL: "https://r2.example.com/avatar/abc.webp?sig=x", publicURL: "https://pub.r2.dev/avatar/abc.webp"}
	handler := upload.PresignedURLHandler(mock)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	req := &upload.PresignedURLRequest{
		Body: upload.PresignedURLRequestBody{FileType: "avatar", SheetUUID: strPtr(uuid.New().String())},
	}

	resp, err := handler(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.Status)
	}
	if resp.Body.UploadURL != mock.uploadURL {
		t.Errorf("unexpected upload_url: %s", resp.Body.UploadURL)
	}
}

func TestPresignedURLHandler_MapBg(t *testing.T) {
	mapUUID := uuid.New()
	mock := &mockR2Client{
		uploadURL: "https://r2.example.com/map_bg/some-file.webp?sig=x",
		publicURL: "https://pub.r2.dev/map_bg/some-file.webp",
	}
	handler := upload.PresignedURLHandler(mock)

	req := &upload.PresignedURLRequest{
		Body: upload.PresignedURLRequestBody{FileType: "map_bg", MapUUID: strPtr(mapUUID.String())},
	}
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	resp, err := handler(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Body.PublicURL != mock.publicURL {
		t.Errorf("unexpected public_url: %s", resp.Body.PublicURL)
	}
	// key must be map_bg/{freshUUID}.webp — NOT the entity UUID
	if mock.capturedKey == "map_bg/"+mapUUID.String()+".webp" {
		t.Error("key must not reuse the entity UUID — breaks CDN cache busting")
	}
}

func TestPresignedURLHandler_UniqueKeysPerUpload(t *testing.T) {
	sheetUUID := uuid.New().String()
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	req := &upload.PresignedURLRequest{
		Body: upload.PresignedURLRequestBody{FileType: "avatar", SheetUUID: strPtr(sheetUUID)},
	}

	mock1 := &mockR2Client{publicURL: "https://pub.r2.dev/avatar/v1.webp"}
	mock2 := &mockR2Client{publicURL: "https://pub.r2.dev/avatar/v2.webp"}

	_, _ = upload.PresignedURLHandler(mock1)(ctx, req)
	_, _ = upload.PresignedURLHandler(mock2)(ctx, req)

	if mock1.capturedKey == mock2.capturedKey {
		t.Errorf("two uploads for the same entity produced the same key %q — CDN cache busting broken", mock1.capturedKey)
	}
}

func TestPresignedURLHandler_MapBg_InvalidUUID(t *testing.T) {
	mock := &mockR2Client{}
	handler := upload.PresignedURLHandler(mock)
	req := &upload.PresignedURLRequest{
		Body: upload.PresignedURLRequestBody{FileType: "map_bg", MapUUID: strPtr("not-a-uuid")},
	}
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	_, err := handler(ctx, req)
	if err == nil {
		t.Fatal("expected error for invalid map_uuid, got nil")
	}
}

func TestPresignedURLHandler_R2Error(t *testing.T) {
	mock := &mockR2Client{err: errors.New("storage unavailable")}
	handler := upload.PresignedURLHandler(mock)
	req := &upload.PresignedURLRequest{
		Body: upload.PresignedURLRequestBody{FileType: "avatar", SheetUUID: strPtr(uuid.New().String())},
	}
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	_, err := handler(ctx, req)
	if err == nil {
		t.Fatal("expected error when r2 client fails, got nil")
	}
}
