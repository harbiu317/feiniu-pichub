package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/levis/pichub/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3 struct {
	client     *minio.Client
	bucket     string
	prefix     string
	publicBase string
	useSSL     bool
	endpoint   string
}

func NewS3(c config.S3Config) (*S3, error) {
	if c.Endpoint == "" || c.Bucket == "" || c.AccessKey == "" || c.SecretKey == "" {
		return nil, fmt.Errorf("S3 配置不完整（endpoint/bucket/access_key/secret_key）")
	}
	endpoint := strings.TrimPrefix(strings.TrimPrefix(c.Endpoint, "https://"), "http://")
	endpoint = strings.TrimRight(endpoint, "/")

	opts := &minio.Options{
		Creds:        credentials.NewStaticV4(c.AccessKey, c.SecretKey, ""),
		Secure:       c.UseSSL,
		Region:       c.Region,
		BucketLookup: minio.BucketLookupAuto,
	}
	if c.PathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	}
	cli, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, err
	}
	return &S3{
		client:     cli,
		bucket:     c.Bucket,
		prefix:     strings.Trim(c.Prefix, "/"),
		publicBase: strings.TrimRight(c.PublicBase, "/"),
		useSSL:     c.UseSSL,
		endpoint:   endpoint,
	}, nil
}

func (s *S3) objKey(key string) string {
	if s.prefix == "" {
		return key
	}
	return s.prefix + "/" + key
}

func (s *S3) Put(key string, data io.Reader, size int64, contentType string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	_, err := s.client.PutObject(ctx, s.bucket, s.objKey(key), data, size,
		minio.PutObjectOptions{ContentType: contentType, CacheControl: "public, max-age=31536000, immutable"})
	if err != nil {
		return wrapS3Err(err, s.bucket, s.objKey(key))
	}
	return nil
}

// TestConnection 执行 HeadBucket 验证凭证和 bucket 可达。
// 返回 nil 表示配置正确。
func (s *S3) TestConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return wrapS3Err(err, s.bucket, "")
	}
	if !exists {
		return fmt.Errorf("bucket %q 不存在或当前凭证无权访问", s.bucket)
	}
	return nil
}

func wrapS3Err(err error, bucket, key string) error {
	if err == nil {
		return nil
	}
	var er minio.ErrorResponse
	if errors.As(err, &er) {
		switch er.Code {
		case "NoSuchBucket":
			return fmt.Errorf("bucket %q 不存在，请核对 bucket 名（腾讯云 COS 需带 AppID 后缀）", bucket)
		case "NoSuchKey":
			return fmt.Errorf("对象键 %q 不存在；此错误出现在上传时通常意味着 bucket 名或 endpoint 格式错误，请检查配置", key)
		case "AccessDenied":
			return fmt.Errorf("访问被拒绝：请检查 AccessKey/SecretKey 是否正确、是否有该 bucket 的读写权限")
		case "SignatureDoesNotMatch":
			return fmt.Errorf("签名不匹配：通常是 SecretKey 错误或 endpoint/region 配置有误")
		case "AuthorizationHeaderMalformed", "InvalidRegion":
			return fmt.Errorf("region 配置错误：请确保 region 与 endpoint 对应的地域一致（%s）", er.Region)
		case "InvalidBucketName":
			return fmt.Errorf("bucket 名称不合法：%q", bucket)
		}
		return fmt.Errorf("S3 错误 [%s] %s", er.Code, er.Message)
	}
	return err
}

func (s *S3) Get(key string) (io.ReadCloser, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	obj, err := s.client.GetObject(ctx, s.bucket, s.objKey(key), minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}
	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, 0, err
	}
	return obj, info.Size, nil
}

func (s *S3) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.client.RemoveObject(ctx, s.bucket, s.objKey(key), minio.RemoveObjectOptions{})
}

func (s *S3) Exists(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := s.client.StatObject(ctx, s.bucket, s.objKey(key), minio.StatObjectOptions{})
	return err == nil
}

func (s *S3) URL(key string) string {
	k := url.PathEscape(s.objKey(key))
	k = strings.ReplaceAll(k, "%2F", "/")
	if s.publicBase != "" {
		return s.publicBase + "/" + k
	}
	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, s.endpoint, s.bucket, k)
}
