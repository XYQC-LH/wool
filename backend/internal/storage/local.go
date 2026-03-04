package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"nexus-api/internal/config"
)

type localStorage struct {
	baseDir              string
	signSecret           []byte
	defaultExpireSeconds int
}

func NewLocalStorage(cfg config.OSSConfig) (ObjectStorage, error) {
	baseDir := strings.TrimSpace(cfg.LocalDir)
	if baseDir == "" {
		return nil, errors.New("STORAGE_LOCAL_DIR 不能为空")
	}

	signSecret := strings.TrimSpace(cfg.SignSecret)
	if signSecret == "" {
		return nil, errors.New("STORAGE_SIGN_SECRET 不能为空")
	}

	expire := cfg.SignExpireSeconds
	if expire <= 0 {
		expire = 900
	}

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}

	return &localStorage{
		baseDir:              baseDir,
		signSecret:           []byte(signSecret),
		defaultExpireSeconds: expire,
	}, nil
}

func ResolveLocalObjectPath(baseDir, objectKey string) (string, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return "", errors.New("baseDir 不能为空")
	}

	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return "", errors.New("objectKey 不能为空")
	}

	objectKey = strings.ReplaceAll(objectKey, "\\", "/")
	objectKey = strings.TrimPrefix(objectKey, "/")
	if objectKey == "" {
		return "", errors.New("objectKey 不能为空")
	}

	if strings.Contains(objectKey, "..") {
		return "", errors.New("objectKey 非法")
	}

	cleaned := path.Clean(objectKey)
	if cleaned == "." || cleaned == "" || strings.HasPrefix(cleaned, "..") {
		return "", errors.New("objectKey 非法")
	}

	fullPath := filepath.Join(baseDir, filepath.FromSlash(cleaned))

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	fullAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(baseAbs, fullAbs)
	if err != nil {
		return "", errors.New("objectKey 非法")
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("objectKey 非法")
	}

	return fullPath, nil
}

func (s *localStorage) PutObject(objectKey string, reader io.Reader, opts PutObjectOptions) error {
	_ = opts
	if reader == nil {
		return errors.New("reader 不能为空")
	}

	fullPath, err := ResolveLocalObjectPath(s.baseDir, objectKey)
	if err != nil {
		return err
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := io.Copy(tmp, reader); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, fullPath)
}

func (s *localStorage) DeleteObject(objectKey string) error {
	fullPath, err := ResolveLocalObjectPath(s.baseDir, objectKey)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *localStorage) SignGetURL(objectKey string, expireSeconds int) (string, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return "", errors.New("objectKey 不能为空")
	}
	objectKey = strings.ReplaceAll(objectKey, "\\", "/")
	objectKey = strings.TrimPrefix(objectKey, "/")

	if expireSeconds <= 0 {
		expireSeconds = s.defaultExpireSeconds
	}

	exp := time.Now().Add(time.Duration(expireSeconds) * time.Second).Unix()
	sig := computeLocalObjectSignature(objectKey, exp, s.signSecret)

	return "/objects/" + objectKey + "?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + sig, nil
}

func computeLocalObjectSignature(objectKey string, exp int64, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(objectKey))
	_, _ = mac.Write([]byte(":"))
	_, _ = mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func VerifyLocalObjectSignature(objectKey string, exp int64, sig string, secret []byte) bool {
	expected := computeLocalObjectSignature(objectKey, exp, secret)
	return hmac.Equal([]byte(expected), []byte(sig))
}
