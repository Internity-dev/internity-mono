// Package storage wraps the MinIO/S3 client: upload with real content-type
// sniffing (never trusting the client's Content-Type header) and presigned
// GET URLs for private buckets. See plan section 2.7 for the bucket layout.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	BucketAvatars     = "internity-avatars"
	BucketAttachments = "internity-attachments"
	BucketDocuments   = "internity-documents"
	BucketLogos       = "internity-logos"
)

type Client struct {
	mc *minio.Client
}

func Open(endpoint, accessKey, secretKey string, useSSL bool) (*Client, error) {
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &Client{mc: mc}, nil
}

// AllowedContentTypes per upload kind — checked against the actual sniffed
// bytes, never the client-supplied header (spec requirement: real file-type
// validation, not a rubber-stamp of whatever the browser claims).
var AllowedContentTypes = map[string][]string{
	"image": {"image/jpeg", "image/png", "image/webp"},
	"pdf":   {"application/pdf"},
}

const (
	MaxImageBytes = 5 << 20  // 5MB
	MaxDocBytes   = 10 << 20 // 10MB
)

type UploadInput struct {
	Bucket           string
	KeyPrefix        string // e.g. "presence/2026/08/22" — date-partitioned, never the client filename
	OriginalFilename string
	Data             []byte
	AllowedKinds     []string // keys into AllowedContentTypes
	MaxBytes         int64
}

type UploadResult struct {
	Key              string
	ContentType      string
	OriginalFilename string
	SizeBytes        int64
}

// Upload validates content-type (sniffed from bytes) and size, then writes
// to a generated {prefix}/{uuid}.{ext} key — never the client's filename, so
// path traversal / overwrite / unicode-spoofing on the original name can't
// reach the storage key (plan section 2.7).
func (c *Client) Upload(ctx context.Context, in UploadInput) (*UploadResult, error) {
	if int64(len(in.Data)) > in.MaxBytes {
		return nil, fmt.Errorf("file exceeds maximum size of %d bytes", in.MaxBytes)
	}

	sniffLen := 512
	if len(in.Data) < sniffLen {
		sniffLen = len(in.Data)
	}
	contentType := http.DetectContentType(in.Data[:sniffLen])

	if !contentTypeAllowed(contentType, in.AllowedKinds) {
		return nil, fmt.Errorf("unsupported file type: %s", contentType)
	}

	ext := extensionFor(contentType)
	key := fmt.Sprintf("%s/%s%s", in.KeyPrefix, uuid.NewString(), ext)

	_, err := c.mc.PutObject(ctx, in.Bucket, key, bytes.NewReader(in.Data), int64(len(in.Data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return nil, err
	}

	return &UploadResult{Key: key, ContentType: contentType, OriginalFilename: in.OriginalFilename, SizeBytes: int64(len(in.Data))}, nil
}

// PresignedGetURL issues a short-lived signed URL — access control happens
// here, not on the bucket: a URL is only ever generated after the caller's
// scope has already been checked by the requesting service (plan 2.7).
func (c *Client) PresignedGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, bucket, key, ttl, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (c *Client) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := c.mc.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return c.mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}

func contentTypeAllowed(contentType string, kinds []string) bool {
	for _, kind := range kinds {
		for _, allowed := range AllowedContentTypes[kind] {
			if contentType == allowed {
				return true
			}
		}
	}
	return false
}

func extensionFor(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}
