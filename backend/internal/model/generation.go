package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// GenerationType 鐢熸垚绫诲瀷
type GenerationType string

const (
	GenerationTypeImage GenerationType = "image"
	GenerationTypeVideo GenerationType = "video"
)

// GenerationStatus 鐢熸垚浠诲姟鐘舵€?
type GenerationStatus string

const (
	GenerationStatusPending    GenerationStatus = "pending"
	GenerationStatusProcessing GenerationStatus = "processing"
	GenerationStatusCompleted  GenerationStatus = "completed"
	GenerationStatusFailed     GenerationStatus = "failed"
	GenerationStatusExpired    GenerationStatus = "expired"
)

// GenerationTask 鐢熸垚浠诲姟
type GenerationTask struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	UserID          uuid.UUID        `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenID         uuid.UUID        `gorm:"type:uuid;index" json:"token_id"`
	Type            GenerationType   `gorm:"type:varchar(20);not null" json:"type"`
	Model           string           `gorm:"type:varchar(100);not null" json:"model"`
	Provider        string           `gorm:"type:varchar(50);not null" json:"provider"`
	Prompt          string           `gorm:"type:text" json:"prompt"`
	Params          JSON             `gorm:"type:jsonb" json:"params"`
	Status          GenerationStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	Progress        float64          `gorm:"default:0" json:"progress"`
	ResultURL       *string          `gorm:"type:text" json:"result_url,omitempty"`
	ResultObjectKey *string          `gorm:"type:text" json:"result_object_key,omitempty"`
	ResultData      JSON             `gorm:"type:jsonb" json:"result_data,omitempty"`
	ErrorMessage    *string          `gorm:"type:text" json:"error_message,omitempty"`
	Cost            decimal.Decimal  `gorm:"type:decimal(10,6);default:0" json:"cost"`
	Duration        int              `gorm:"default:0" json:"duration"` // 姣
	CreatedAt       time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
	CompletedAt     *time.Time       `json:"completed_at,omitempty"`
	ExpiresAt       *time.Time       `json:"expires_at,omitempty"`

	// 鍏宠仈
	User  *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Token *Token `gorm:"foreignKey:TokenID" json:"token,omitempty"`
}

// TableName 琛ㄥ悕
func (GenerationTask) TableName() string {
	return "generation_tasks"
}

// ImageGenerationRequest 鍥剧墖鐢熸垚璇锋眰
type ImageGenerationRequest struct {
	Model          string   `json:"model" binding:"required"`
	Prompt         string   `json:"prompt" binding:"required"`
	N              int      `json:"n,omitempty"`
	Size           string   `json:"size,omitempty"`
	AspectRatio    string   `json:"aspect_ratio,omitempty"`
	Resolution     string   `json:"resolution,omitempty"`
	ResponseFormat string   `json:"response_format,omitempty"`
	User           string   `json:"user,omitempty"`
	URLs           []string `json:"urls,omitempty"`
	Image          string   `json:"image,omitempty"`
	Seed           int      `json:"seed,omitempty"`
	Watermark      bool     `json:"watermark,omitempty"`
}

// ImageGenerationResponse 鍥剧墖鐢熸垚鍝嶅簲
type ImageGenerationResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
	TaskID  string      `json:"task_id,omitempty"`
}

// ImageData 鍥剧墖鏁版嵁
type ImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

// VideoGenerationRequest 瑙嗛鐢熸垚璇锋眰
type VideoGenerationRequest struct {
	Model       string `json:"model" binding:"required"`
	Prompt      string `json:"prompt" binding:"required"`
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Duration    int    `json:"duration,omitempty"`
	Size        string `json:"size,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	User        string `json:"user,omitempty"`
}

// VideoGenerationResponse 瑙嗛鐢熸垚鍝嶅簲
type VideoGenerationResponse struct {
	ID        string           `json:"id"`
	Status    GenerationStatus `json:"status"`
	Progress  float64          `json:"progress"`
	CreatedAt int64            `json:"created_at"`
	Data      *VideoData       `json:"data,omitempty"`
	Error     *string          `json:"error,omitempty"`
}

// VideoData 瑙嗛鏁版嵁
type VideoData struct {
	URL      string `json:"url,omitempty"`
	Duration int    `json:"duration,omitempty"`
}

// GenerationTaskResponse 鐢熸垚浠诲姟鍝嶅簲
type GenerationTaskResponse struct {
	ID           string           `json:"id"`
	Type         GenerationType   `json:"type"`
	Model        string           `json:"model"`
	Status       GenerationStatus `json:"status"`
	Progress     float64          `json:"progress"`
	ResultURL    *string          `json:"result_url,omitempty"`
	ErrorMessage *string          `json:"error_message,omitempty"`
	Cost         decimal.Decimal  `json:"cost"`
	Duration     int              `json:"duration"`
	CreatedAt    time.Time        `json:"created_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
}

// ToResponse 杞崲涓哄搷搴旂粨鏋?
func (t *GenerationTask) ToResponse() *GenerationTaskResponse {
	return &GenerationTaskResponse{
		ID:           t.ID.String(),
		Type:         t.Type,
		Model:        t.Model,
		Status:       t.Status,
		Progress:     t.Progress,
		ResultURL:    t.ResultURL,
		ErrorMessage: t.ErrorMessage,
		Cost:         t.Cost,
		Duration:     t.Duration,
		CreatedAt:    t.CreatedAt,
		CompletedAt:  t.CompletedAt,
	}
}

// ⚠️ 不再维护硬编码的 model → provider 映射。
// 多模态调度以 RouteKey=(operation, model_id) 为核心，源头由 model_providers 配置决定。
