package upload

import (
	"context"
	"net/http"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/r2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type PresignResult = r2.PresignResult

// IR2Client is the interface for generating presigned PUT URLs against R2.
type IR2Client interface {
	NewPresignedPutURL(ctx context.Context, key string, ttl time.Duration) (PresignResult, error)
}

// PresignedURLRequestBody carries the JSON body of the presigned-URL request.
type PresignedURLRequestBody struct {
	FileType  string  `json:"file_type" doc:"'avatar', 'cover', or 'map_bg'"`
	SheetUUID *string `json:"sheet_uuid,omitempty"`
	MapUUID   *string `json:"map_uuid,omitempty"`
}

// PresignedURLRequest is the huma input type for the presigned-URL endpoint.
type PresignedURLRequest struct {
	Body PresignedURLRequestBody
}

// PresignedURLResponseBody carries the JSON body of the presigned-URL response.
type PresignedURLResponseBody struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
}

// PresignedURLResponse is the huma output type for the presigned-URL endpoint.
type PresignedURLResponse struct {
	Body   PresignedURLResponseBody
	Status int
}

// PresignedURLHandler returns a huma handler that generates a presigned PUT URL
// for direct browser-to-R2 uploads of avatar or cover images.
func PresignedURLHandler(
	r2Client IR2Client,
) func(context.Context, *PresignedURLRequest) (*PresignedURLResponse, error) {
	return func(ctx context.Context, req *PresignedURLRequest) (*PresignedURLResponse, error) {
		_, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		fileType := req.Body.FileType
		var key string
		switch fileType {
		case "avatar", "cover":
			if req.Body.SheetUUID == nil {
				return nil, huma.Error400BadRequest("sheet_uuid is required for avatar/cover")
			}
			sheetUUID, err := uuid.Parse(*req.Body.SheetUUID)
			if err != nil {
				return nil, huma.Error400BadRequest("invalid sheet_uuid")
			}
			key = fileType + "/" + sheetUUID.String() + ".webp"
		case "map_bg":
			if req.Body.MapUUID == nil {
				return nil, huma.Error400BadRequest("map_uuid is required for map_bg")
			}
			mapUUID, err := uuid.Parse(*req.Body.MapUUID)
			if err != nil {
				return nil, huma.Error400BadRequest("invalid map_uuid")
			}
			key = "map_bg/" + mapUUID.String() + ".webp"
		default:
			return nil, huma.Error422UnprocessableEntity("file_type must be 'avatar', 'cover', or 'map_bg'")
		}
		result, err := r2Client.NewPresignedPutURL(ctx, key, 5*time.Minute)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to generate upload URL")
		}

		return &PresignedURLResponse{
			Body:   PresignedURLResponseBody{UploadURL: result.UploadURL, PublicURL: result.PublicURL},
			Status: http.StatusOK,
		}, nil
	}
}
