package r2

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	presign   *s3.PresignClient
	bucket    string
	publicURL string
}

func NewClient(accountID, accessKeyID, secretAccessKey, bucket, publicURL string) *Client {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg := aws.Config{
		Region: "auto",
		Credentials: credentials.NewStaticCredentialsProvider(
			accessKeyID, secretAccessKey, "",
		),
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &Client{
		presign:   s3.NewPresignClient(s3Client),
		bucket:    bucket,
		publicURL: publicURL,
	}
}

type PresignResult struct {
	UploadURL string
	PublicURL string
}

// NewPresignedPutURL gera uma presigned PUT URL para upload direto no R2.
// key exemplo: "avatar/abc123.webp"
func (c *Client) NewPresignedPutURL(ctx context.Context, key string, ttl time.Duration) (PresignResult, error) {
	req, err := c.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String("image/webp"),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return PresignResult{}, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return PresignResult{
		UploadURL: req.URL,
		PublicURL: fmt.Sprintf("%s/%s", c.publicURL, key),
	}, nil
}
