package storage

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Options covers AWS S3 and S3-compatible services (MinIO, Aliyun OSS,
// Tencent COS, Cloudflare R2, ...).
type S3Options struct {
	Endpoint  string // host[:port], without scheme
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	// PathStyle forces bucket-in-path addressing, needed by MinIO.
	PathStyle bool
	// Prefix namespaces every key inside the bucket.
	Prefix string
	// PublicBaseURL is a CDN or public-bucket address. When set, URL() returns
	// plain links under it; otherwise URL() returns time-limited presigned links.
	PublicBaseURL string
}

type S3 struct {
	client *minio.Client
	opts   S3Options
}

func NewS3(opts S3Options) (*S3, error) {
	if opts.Endpoint == "" || opts.Bucket == "" {
		return nil, errors.New("storage: s3 endpoint and bucket are required")
	}
	lookup := minio.BucketLookupAuto
	if opts.PathStyle {
		lookup = minio.BucketLookupPath
	}
	client, err := minio.New(opts.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, ""),
		Secure:       opts.UseSSL,
		Region:       opts.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, err
	}
	opts.Prefix = strings.Trim(opts.Prefix, "/")
	opts.PublicBaseURL = strings.TrimRight(opts.PublicBaseURL, "/")
	return &S3{client: client, opts: opts}, nil
}

func (s *S3) Name() string { return "s3" }

func (s *S3) objectKey(key string) (string, error) {
	clean, err := cleanKey(key)
	if err != nil {
		return "", err
	}
	if s.opts.Prefix != "" {
		clean = s.opts.Prefix + "/" + clean
	}
	return clean, nil
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	k, err := s.objectKey(key)
	if err != nil {
		return err
	}
	if size < 0 {
		size = -1
	}
	_, err = s.client.PutObject(ctx, s.opts.Bucket, k, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *S3) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	k, err := s.objectKey(key)
	if err != nil {
		return nil, err
	}
	obj, err := s.client.GetObject(ctx, s.opts.Bucket, k, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	// GetObject is lazy; Stat surfaces a missing object up front.
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return obj, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	k, err := s.objectKey(key)
	if err != nil {
		return err
	}
	// S3 deletes are idempotent: removing a missing object is not an error.
	return s.client.RemoveObject(ctx, s.opts.Bucket, k, minio.RemoveObjectOptions{})
}

func (s *S3) URL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	k, err := s.objectKey(key)
	if err != nil {
		return "", err
	}
	if s.opts.PublicBaseURL != "" {
		return s.opts.PublicBaseURL + "/" + k, nil
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	u, err := s.client.PresignedGetObject(ctx, s.opts.Bucket, k, ttl, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
