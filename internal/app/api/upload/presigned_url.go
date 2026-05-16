package upload

import (
	"context"
	"net/http"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// PresignResult holds the URLs returned by the R2 presign operation.
type PresignResult struct {
	UploadURL string
	PublicURL string
}

// IR2Client is the interface for generating presigned PUT URLs against R2.
type IR2Client interface {
	NewPresignedPutURL(ctx context.Context, key string, ttl time.Duration) (PresignResult, error)
}

// PresignedURLRequestBody carries the JSON body of the presigned-URL request.
type PresignedURLRequestBody struct {
	FileType  string `json:"file_type" doc:"'avatar' or 'cover'"`
	SheetUUID string `json:"sheet_uuid"`
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
		if fileType != "avatar" && fileType != "cover" {
			return nil, huma.Error422UnprocessableEntity("file_type must be 'avatar' or 'cover'")
		}

		sheetUUID, err := uuid.Parse(req.Body.SheetUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid sheet_uuid")
		}

		key := fileType + "/" + sheetUUID.String() + ".webp"
		result, err := r2Client.NewPresignedPutURL(ctx, key, 5*time.Minute)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}

		return &PresignedURLResponse{
			Body:   PresignedURLResponseBody{UploadURL: result.UploadURL, PublicURL: result.PublicURL},
			Status: http.StatusOK,
		}, nil
	}
}
