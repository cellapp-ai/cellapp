package hosting

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage interface {
	Put(context.Context, string, io.Reader, int64) error
	Get(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
	Health(context.Context) error
}
type S3Storage struct {
	client *minio.Client
	bucket string
}

func NewStorage(c Config) (*S3Storage, error) {
	u, e := url.Parse(c.S3Endpoint)
	if e != nil {
		return nil, e
	}
	if u.Host == "" || u.Path != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, fmt.Errorf("invalid S3 endpoint")
	}
	if c.Production && u.Scheme != "https" {
		return nil, fmt.Errorf("production S3 endpoint must use HTTPS")
	}
	client, e := minio.New(u.Host, &minio.Options{Creds: credentials.NewStaticV4(c.QiniuAccessKey, c.QiniuSecretKey, ""), Secure: u.Scheme == "https", Region: c.S3Region, BucketLookup: minio.BucketLookupPath})
	return &S3Storage{client, c.S3Bucket}, e
}
func (s *S3Storage) Put(ctx context.Context, key string, r io.Reader, n int64) error {
	_, e := s.client.PutObject(ctx, s.bucket, key, r, n, minio.PutObjectOptions{ContentType: "application/octet-stream", DisableMultipart: true, SendContentMd5: true})
	return e
}
func (s *S3Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	o, e := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if e != nil {
		return nil, e
	}
	if _, e = o.Stat(); e != nil {
		_ = o.Close()
		return nil, e
	}
	return o, nil
}
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}
func (s *S3Storage) Health(ctx context.Context) error {
	exists, e := s.client.BucketExists(ctx, s.bucket)
	if e != nil {
		return e
	}
	if !exists {
		return fmt.Errorf("configured S3 bucket does not exist")
	}
	return nil
}
