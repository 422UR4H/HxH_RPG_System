package upload_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/upload"
	"github.com/google/uuid"
)

type mockR2Client struct {
	uploadURL string
	publicURL string
	err       error
}

func (m *mockR2Client) NewPresignedPutURL(_ context.Context, key string, _ time.Duration) (upload.PresignResult, error) {
	return upload.PresignResult{UploadURL: m.uploadURL, PublicURL: m.publicURL}, m.err
}

func TestPresignedURLHandler_InvalidFileType(t *testing.T) {
	mock := &mockR2Client{}
	handler := upload.PresignedURLHandler(mock)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	req := &upload.PresignedURLRequest{
		Body: upload.PresignedURLRequestBody{FileType: "invalid", SheetUUID: uuid.New().String()},
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
	sheetUUID := uuid.New().String()
	req := &upload.PresignedURLRequest{
		Body: upload.PresignedURLRequestBody{FileType: "avatar", SheetUUID: sheetUUID},
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
