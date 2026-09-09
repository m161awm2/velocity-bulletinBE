package upload

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

var ErrDisabled = errors.New("image uploads are not configured")

type PresignedUpload struct {
	UploadURL string            `json:"uploadUrl"`
	ObjectKey string            `json:"objectKey"`
	PublicURL string            `json:"publicUrl"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expiresAt"`
}

type Service struct {
	bucket        string
	publicBaseURL string
	presigner     *s3.PresignClient
}

func New(ctx context.Context, region, bucket, publicBaseURL string) (*Service, error) {
	service := &Service{bucket: bucket, publicBaseURL: strings.TrimRight(publicBaseURL, "/")}
	if bucket == "" {
		return service, nil
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	service.presigner = s3.NewPresignClient(s3.NewFromConfig(awsCfg))
	return service, nil
}

func (s *Service) Presign(ctx context.Context, filename, contentType string, size int64) (*PresignedUpload, error) {
	if s.presigner == nil || s.bucket == "" {
		return nil, ErrDisabled
	}
	if size <= 0 || size > 5*1024*1024 {
		return nil, errors.New("file size must be between 1 byte and 5 MB")
	}
	allowed := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}
	ext, ok := allowed[strings.ToLower(contentType)]
	if !ok {
		return nil, errors.New("only JPEG, PNG, and WebP images are supported")
	}
	if detected := strings.ToLower(filepath.Ext(filename)); detected != "" {
		canonical := strings.ToLower(mime.TypeByExtension(detected))
		if canonical != "" && canonical != strings.ToLower(contentType) && !(detected == ".jpeg" && contentType == "image/jpeg") {
			return nil, errors.New("filename extension does not match content type")
		}
	}
	objectKey := "media/" + uuid.NewString() + ext
	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	request, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(objectKey), ContentType: aws.String(contentType), ContentLength: aws.Int64(size),
	}, s3.WithPresignExpires(10*time.Minute))
	if err != nil {
		return nil, fmt.Errorf("presign upload: %w", err)
	}
	publicURL := objectKey
	if s.publicBaseURL != "" {
		publicURL = s.publicBaseURL + "/" + objectKey
	}
	return &PresignedUpload{
		UploadURL: request.URL, ObjectKey: objectKey, PublicURL: publicURL,
		Headers: map[string]string{"Content-Type": contentType}, ExpiresAt: expiresAt,
	}, nil
}
