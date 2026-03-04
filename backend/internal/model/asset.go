package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AssetOwnerType 资源归属类型
type AssetOwnerType string

const (
	AssetOwnerTypeSite AssetOwnerType = "site" // 网站素材（公共可访问，但仍通过签名URL读取私有桶）
	AssetOwnerTypeUser AssetOwnerType = "user" // 用户私有资源
)

// AssetPurpose 资源用途
type AssetPurpose string

const (
	AssetPurposeSiteMaterial AssetPurpose = "site_material"
	AssetPurposeUserUpload   AssetPurpose = "user_upload"
	AssetPurposeAPIOutput    AssetPurpose = "api_output"
)

// AssetKind 资源类型（用于Key前缀与策略区分）
type AssetKind string

const (
	AssetKindImage AssetKind = "img"
	AssetKindVideo AssetKind = "video"
	AssetKindFile  AssetKind = "file"
)

// AssetStatus 资源状态
type AssetStatus string

const (
	AssetStatusActive  AssetStatus = "active"
	AssetStatusDeleted AssetStatus = "deleted"
)

// Asset 资源元数据（对象存储为事实存储，DB仅存索引与权限语义）
type Asset struct {
	ID               uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OwnerType        AssetOwnerType `gorm:"type:varchar(20);not null;index" json:"owner_type"`
	UserID           *uuid.UUID     `gorm:"type:uuid;index" json:"user_id,omitempty"`
	Purpose          AssetPurpose   `gorm:"type:varchar(20);not null;index" json:"purpose"`
	Kind             AssetKind      `gorm:"type:varchar(10);not null;index" json:"kind"`
	Bucket           string         `gorm:"type:varchar(100);not null" json:"bucket"`
	ObjectKey        string         `gorm:"type:varchar(1024);not null;uniqueIndex" json:"object_key"`
	OriginalFilename string         `gorm:"type:varchar(255)" json:"original_filename,omitempty"`
	MimeType         string         `gorm:"type:varchar(200)" json:"mime_type,omitempty"`
	SizeBytes        int64          `gorm:"not null;default:0" json:"size_bytes"`
	SHA256           *string        `gorm:"type:char(64)" json:"sha256,omitempty"`
	ExpiresAt        *time.Time     `json:"expires_at,omitempty"`
	Status           AssetStatus    `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	CreatedAt        time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt        *time.Time     `json:"deleted_at,omitempty"`
}

func (Asset) TableName() string {
	return "assets"
}

func (a *Asset) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (a *Asset) IsPublic() bool {
	return a.OwnerType == AssetOwnerTypeSite
}
