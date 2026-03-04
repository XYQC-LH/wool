package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"path"
	"strings"
	"time"

	"nexus-api/internal/config"
	"nexus-api/internal/model"
	"nexus-api/internal/repository"
	storage "nexus-api/internal/storage"

	"github.com/google/uuid"
)

// UploadInput 上传输入
type UploadInput struct {
	Reader           io.Reader
	OriginalFilename string
	ContentType      string
	SizeBytes        int64
}

// AssetService 资源服务接口
type AssetService interface {
	UploadSiteMaterial(input UploadInput) (*model.Asset, error)
	UploadUserUpload(userID uuid.UUID, input UploadInput) (*model.Asset, error)
	UploadUserOutput(userID uuid.UUID, input UploadInput) (*model.Asset, error)
	GetByID(id uuid.UUID) (*model.Asset, error)
	SignGetURL(asset *model.Asset, expireSeconds int) (string, error)
}

type assetService struct {
	cfg       *config.Config
	assetRepo repository.AssetRepository
	objStore  storage.ObjectStorage
	now       func() time.Time
}

func NewAssetService(cfg *config.Config, assetRepo repository.AssetRepository, objStore storage.ObjectStorage) AssetService {
	return &assetService{
		cfg:       cfg,
		assetRepo: assetRepo,
		objStore:  objStore,
		now:       time.Now,
	}
}

func (s *assetService) UploadSiteMaterial(input UploadInput) (*model.Asset, error) {
	return s.upload(input, model.AssetOwnerTypeSite, nil, model.AssetPurposeSiteMaterial)
}

func (s *assetService) UploadUserUpload(userID uuid.UUID, input UploadInput) (*model.Asset, error) {
	return s.upload(input, model.AssetOwnerTypeUser, &userID, model.AssetPurposeUserUpload)
}

func (s *assetService) UploadUserOutput(userID uuid.UUID, input UploadInput) (*model.Asset, error) {
	return s.upload(input, model.AssetOwnerTypeUser, &userID, model.AssetPurposeAPIOutput)
}

func (s *assetService) GetByID(id uuid.UUID) (*model.Asset, error) {
	return s.assetRepo.GetByID(id)
}

func (s *assetService) SignGetURL(asset *model.Asset, expireSeconds int) (string, error) {
	if asset == nil {
		return "", errors.New("asset 不能为空")
	}
	if asset.Status != model.AssetStatusActive {
		return "", errors.New("资源已失效")
	}
	return s.objStore.SignGetURL(asset.ObjectKey, expireSeconds)
}

func (s *assetService) upload(input UploadInput, ownerType model.AssetOwnerType, userID *uuid.UUID, purpose model.AssetPurpose) (*model.Asset, error) {
	if s.cfg == nil {
		return nil, errors.New("配置未初始化")
	}
	if s.assetRepo == nil {
		return nil, errors.New("资源仓储未初始化")
	}
	if s.objStore == nil {
		return nil, errors.New("对象存储未初始化")
	}
	if strings.TrimSpace(s.cfg.OSS.Bucket) == "" {
		return nil, errors.New("OSS_BUCKET 未配置")
	}
	if input.Reader == nil {
		return nil, errors.New("上传内容为空")
	}

	now := s.now()
	kind := detectAssetKind(input.ContentType, input.OriginalFilename)

	var objectKey string
	switch {
	case ownerType == model.AssetOwnerTypeSite && purpose == model.AssetPurposeSiteMaterial:
		objectKey = BuildSiteObjectKey(kind, input.OriginalFilename, now)
	case ownerType == model.AssetOwnerTypeUser && userID != nil && purpose == model.AssetPurposeUserUpload:
		objectKey = BuildUserUploadObjectKey(*userID, kind, input.OriginalFilename, now)
	case ownerType == model.AssetOwnerTypeUser && userID != nil && purpose == model.AssetPurposeAPIOutput:
		objectKey = BuildUserOutputObjectKey(*userID, kind, input.OriginalFilename, now)
	default:
		return nil, errors.New("不支持的资源类型")
	}

	contentType := strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	hasher := sha256.New()
	tee := io.TeeReader(input.Reader, hasher)

	if err := s.objStore.PutObject(objectKey, tee, storage.PutObjectOptions{
		ContentType:   contentType,
		ContentLength: input.SizeBytes,
	}); err != nil {
		return nil, err
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	originalFilename := sanitizeOriginalFilename(input.OriginalFilename)

	asset := &model.Asset{
		OwnerType:        ownerType,
		UserID:           userID,
		Purpose:          purpose,
		Kind:             kind,
		Bucket:           s.cfg.OSS.Bucket,
		ObjectKey:        objectKey,
		OriginalFilename: originalFilename,
		MimeType:         contentType,
		SizeBytes:        input.SizeBytes,
		SHA256:           &sum,
		Status:           model.AssetStatusActive,
	}

	if err := s.assetRepo.Create(asset); err != nil {
		_ = s.objStore.DeleteObject(objectKey)
		return nil, err
	}

	return asset, nil
}

func sanitizeOriginalFilename(originalFilename string) string {
	originalFilename = strings.TrimSpace(originalFilename)
	if originalFilename == "" {
		return ""
	}
	filename := path.Base(strings.ReplaceAll(originalFilename, "\\", "/"))
	if filename == "." || filename == "/" {
		return ""
	}
	return filename
}

func detectAssetKind(contentType, originalFilename string) model.AssetKind {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.HasPrefix(ct, "image/"):
		return model.AssetKindImage
	case strings.HasPrefix(ct, "video/"):
		return model.AssetKindVideo
	}

	ext := strings.ToLower(strings.TrimPrefix(path.Ext(path.Base(strings.ReplaceAll(originalFilename, "\\", "/"))), "."))
	switch ext {
	case "png", "jpg", "jpeg", "gif", "webp", "bmp", "svg":
		return model.AssetKindImage
	case "mp4", "webm", "mov", "mkv", "avi":
		return model.AssetKindVideo
	default:
		return model.AssetKindFile
	}
}
