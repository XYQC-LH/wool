package storage

import (
	"errors"
	"strings"

	"nexus-api/internal/config"
)

func NewObjectStorage(cfg config.OSSConfig) (ObjectStorage, error) {
	driver := strings.ToLower(strings.TrimSpace(cfg.Driver))
	switch driver {
	case "", "oss":
		return NewOSSStorage(cfg)
	case "local":
		return NewLocalStorage(cfg)
	default:
		return nil, errors.New("不支持的 STORAGE_DRIVER: " + cfg.Driver)
	}
}
