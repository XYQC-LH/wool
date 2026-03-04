package storage

import (
	"errors"
	"io"
	"net/url"
	"strings"

	"nexus-api/internal/config"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// PutObjectOptions PutObject 参数
type PutObjectOptions struct {
	ContentType   string
	ContentLength int64
}

// ObjectStorage 对象存储抽象
type ObjectStorage interface {
	PutObject(objectKey string, reader io.Reader, opts PutObjectOptions) error
	DeleteObject(objectKey string) error
	SignGetURL(objectKey string, expireSeconds int) (string, error)
}

type ossStorage struct {
	bucket               *oss.Bucket
	defaultExpireSeconds int
}

func NewOSSStorage(cfg config.OSSConfig) (ObjectStorage, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, errors.New("OSS_ENDPOINT 不能为空")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, errors.New("OSS_BUCKET 不能为空")
	}
	if strings.TrimSpace(cfg.AccessKeyID) == "" {
		return nil, errors.New("OSS_ACCESS_KEY_ID 不能为空")
	}
	if strings.TrimSpace(cfg.AccessKeySecret) == "" {
		return nil, errors.New("OSS_ACCESS_KEY_SECRET 不能为空")
	}

	endpoint := normalizeEndpoint(cfg.Endpoint)
	useCname := false

	if strings.TrimSpace(cfg.PublicBaseURL) != "" {
		if u, err := url.Parse(normalizeEndpoint(cfg.PublicBaseURL)); err == nil && u.Host != "" {
			endpoint = u.Scheme + "://" + u.Host
			useCname = true
		}
	}

	opts := make([]oss.ClientOption, 0, 1)
	if useCname {
		opts = append(opts, oss.UseCname(true))
	}

	client, err := oss.New(endpoint, cfg.AccessKeyID, cfg.AccessKeySecret, opts...)
	if err != nil {
		return nil, err
	}

	bucket, err := client.Bucket(cfg.Bucket)
	if err != nil {
		return nil, err
	}

	expire := cfg.SignExpireSeconds
	if expire <= 0 {
		expire = 900
	}

	return &ossStorage{
		bucket:               bucket,
		defaultExpireSeconds: expire,
	}, nil
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	return "https://" + endpoint
}

func (s *ossStorage) PutObject(objectKey string, reader io.Reader, opts PutObjectOptions) error {
	if strings.TrimSpace(objectKey) == "" {
		return errors.New("objectKey 不能为空")
	}

	options := make([]oss.Option, 0, 2)
	if strings.TrimSpace(opts.ContentType) != "" {
		options = append(options, oss.ContentType(opts.ContentType))
	}
	if opts.ContentLength > 0 {
		options = append(options, oss.ContentLength(opts.ContentLength))
	}

	return s.bucket.PutObject(objectKey, reader, options...)
}

func (s *ossStorage) DeleteObject(objectKey string) error {
	if strings.TrimSpace(objectKey) == "" {
		return errors.New("objectKey 不能为空")
	}
	return s.bucket.DeleteObject(objectKey)
}

func (s *ossStorage) SignGetURL(objectKey string, expireSeconds int) (string, error) {
	if strings.TrimSpace(objectKey) == "" {
		return "", errors.New("objectKey 不能为空")
	}

	if expireSeconds <= 0 {
		expireSeconds = s.defaultExpireSeconds
	}

	return s.bucket.SignURL(objectKey, oss.HTTPGet, int64(expireSeconds))
}
