package upload_test

import (
	"context"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/upload"
	"github.com/google/uuid"
)

func TestPresignedURLHandler_ErrorsReadInPortuguese(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	handler := upload.PresignedURLHandler(&mockR2Client{})

	_, err := handler(ctx, &upload.PresignedURLRequest{Body: upload.PresignedURLRequestBody{FileType: "gif"}})
	if err == nil || err.Error() != "Tipo de imagem inválido: escolha avatar, capa ou fundo do mapa." {
		t.Errorf("invalid type: err = %v", err)
	}

	_, err = handler(ctx, &upload.PresignedURLRequest{Body: upload.PresignedURLRequestBody{FileType: "avatar"}})
	if err == nil || err.Error() != "Escolha a ficha da imagem." {
		t.Errorf("missing sheet: err = %v", err)
	}
}
