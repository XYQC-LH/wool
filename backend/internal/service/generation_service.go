package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"nexus-api/internal/config"
	"nexus-api/internal/model"
	"nexus-api/internal/repository"
	"nexus-api/internal/service/scheduler"
	"nexus-api/internal/storage"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrGenerationTaskNotFound  = errors.New("generation task not found")
	ErrGenerationTaskForbidden = errors.New("forbidden")
	ErrIdempotencyConflict     = errors.New("idempotency conflict")
)

// GenerationService 閻㈢喐鍨氶張宥呭閹恒儱褰?
type GenerationService interface {
	GenerateImage(ctx context.Context, req *model.ImageGenerationRequest, token *model.Token, idempotencyKey string) (*model.ImageGenerationResponse, error)
	CreateImageTask(ctx context.Context, req *model.ImageGenerationRequest, token *model.Token, idempotencyKey string) (*model.ImageGenerationResponse, error)
	GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest, token *model.Token, idempotencyKey string) (*model.VideoGenerationResponse, error)
	GetTaskStatus(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) (*model.GenerationTaskResponse, error)
	ListUserTasks(ctx context.Context, userID uuid.UUID, taskType model.GenerationType, page, pageSize int) ([]model.GenerationTaskResponse, int64, error)
}

type generationService struct {
	generationRepo    repository.GenerationRepository
	idempotencyRepo   repository.IdempotencyKeyRepository
	pricingRuleRepo   repository.ProviderPricingRuleRepository
	logRepo           repository.LogRepository
	userRepo          repository.UserRepository
	modelRepo         repository.ModelRepository
	providerRepo      repository.ModelProviderRepository
	tokenService      TokenService
	cascadeController scheduler.CascadeController
	commitGuard       scheduler.CommitGuard

	httpClient        *http.Client
	resourcePoolURL   string
	objStore          storage.ObjectStorage
	signExpireSeconds int
}

// NewGenerationService 閸掓稑缂撻悽鐔稿灇閺堝秴濮?
func NewGenerationService(
	generationRepo repository.GenerationRepository,
	idempotencyRepo repository.IdempotencyKeyRepository,
	pricingRuleRepo repository.ProviderPricingRuleRepository,
	logRepo repository.LogRepository,
	userRepo repository.UserRepository,
	modelRepo repository.ModelRepository,
	providerRepo repository.ModelProviderRepository,
	tokenService TokenService,
	cascadeController scheduler.CascadeController,
	commitGuard scheduler.CommitGuard,
	objStore storage.ObjectStorage,
	cfg *config.Config,
) GenerationService {
	resourcePoolURL := "http://localhost:8001"
	timeout := 10 * time.Minute
	signExpireSeconds := 900

	if cfg != nil && cfg.ResourcePool.URL != "" {
		resourcePoolURL = cfg.ResourcePool.URL
	}
	if cfg != nil && cfg.ResourcePool.Timeout > 0 {
		timeout = time.Duration(cfg.ResourcePool.Timeout) * time.Second
	}
	if cfg != nil && cfg.OSS.SignExpireSeconds > 0 {
		signExpireSeconds = cfg.OSS.SignExpireSeconds
	}

	return &generationService{
		generationRepo:    generationRepo,
		idempotencyRepo:   idempotencyRepo,
		pricingRuleRepo:   pricingRuleRepo,
		logRepo:           logRepo,
		userRepo:          userRepo,
		modelRepo:         modelRepo,
		providerRepo:      providerRepo,
		tokenService:      tokenService,
		cascadeController: cascadeController,
		commitGuard:       commitGuard,
		resourcePoolURL:   resourcePoolURL,
		objStore:          objStore,
		signExpireSeconds: signExpireSeconds,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (s *generationService) resolvePricingRulePrice(ctx context.Context, providerID uint, operation string, unit string) (decimal.Decimal, bool) {
	if s == nil || s.pricingRuleRepo == nil {
		return decimal.Zero, false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	unit = strings.TrimSpace(unit)
	if unit == "" {
		return decimal.Zero, false
	}

	rule, err := s.pricingRuleRepo.GetByProviderOperationUnit(ctx, providerID, operation, unit)
	if err != nil || rule == nil {
		return decimal.Zero, false
	}
	if !rule.Enabled {
		return decimal.Zero, false
	}
	return rule.PricePerUnit, true
}

func (s *generationService) resolveCharge(ctx context.Context, providerID uint, operation string, primaryUnit string, quantity int64) (decimal.Decimal, bool) {
	if quantity <= 0 {
		quantity = 1
	}

	if charged, ok := s.resolvePricingRulePrice(ctx, providerID, operation, primaryUnit); ok {
		return charged.Mul(decimal.NewFromInt(quantity)), true
	}

	if charged, ok := s.resolvePricingRulePrice(ctx, providerID, operation, "request"); ok {
		return charged, true
	}

	return decimal.Zero, false
}

func (s *generationService) resolvePricingRuleCost(ctx context.Context, providerID uint, operation string, unit string) (decimal.Decimal, bool) {
	if s == nil || s.pricingRuleRepo == nil || providerID == 0 {
		return decimal.Zero, false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	unit = strings.TrimSpace(unit)
	if unit == "" {
		return decimal.Zero, false
	}

	rule, err := s.pricingRuleRepo.GetByProviderOperationUnit(ctx, providerID, operation, unit)
	if err != nil || rule == nil || !rule.Enabled {
		return decimal.Zero, false
	}
	if rule.CostPerUnit.LessThan(decimal.Zero) {
		return decimal.Zero, false
	}
	return rule.CostPerUnit, true
}

func (s *generationService) resolveUpstreamCost(ctx context.Context, providerID uint, operation string, primaryUnit string, quantity int64) (decimal.Decimal, bool) {
	if quantity <= 0 {
		quantity = 1
	}

	if unitCost, ok := s.resolvePricingRuleCost(ctx, providerID, operation, primaryUnit); ok {
		return unitCost.Mul(decimal.NewFromInt(quantity)), true
	}
	if unitCost, ok := s.resolvePricingRuleCost(ctx, providerID, operation, "request"); ok {
		return unitCost, true
	}
	return decimal.Zero, false
}

func (s *generationService) createGenerationLog(
	token *model.Token,
	provider *model.ModelProvider,
	modelID string,
	operation string,
	taskID uuid.UUID,
	cost decimal.Decimal,
	upstreamCost decimal.Decimal,
	latencyMs int,
	status model.LogStatus,
	errMsg string,
) {
	if s == nil || s.logRepo == nil || token == nil {
		return
	}

	logItem := &model.Log{
		ID:           uuid.New(),
		UserID:       token.UserID,
		TokenID:      token.ID,
		Model:        modelID,
		TotalCost:    cost,
		UpstreamCost: upstreamCost,
		Duration:     latencyMs,
		Status:       status,
		IsStream:     false,
		ErrorMessage: errMsg,
		Metadata: model.JSON{
			"operation":          operation,
			"modality":           "generation",
			"generation_task_id": taskID.String(),
		},
	}

	if provider != nil && provider.Channel != nil {
		logItem.ChannelID = provider.Channel.ID
	}

	_ = s.logRepo.Create(logItem)
}

func normalizeIdempotencyKey(raw string) string {
	key := strings.TrimSpace(raw)
	if key == "" {
		return ""
	}
	if len(key) > 128 {
		key = key[:128]
	}
	return key
}

var nonHexRegexp = regexp.MustCompile(`[^a-f0-9]`)

func hashRequest(payload any) string {
	b, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	hex := fmt.Sprintf("%x", sum[:])
	return nonHexRegexp.ReplaceAllString(hex, "")
}

func (s *generationService) buildImageResponseFromTask(task *model.GenerationTask) *model.ImageGenerationResponse {
	if task == nil {
		return nil
	}

	resp := &model.ImageGenerationResponse{
		Created: task.CreatedAt.Unix(),
		TaskID:  task.ID.String(),
		Data:    []model.ImageData{},
	}

	url := ""
	if task.ResultURL != nil {
		url = strings.TrimSpace(*task.ResultURL)
	}
	if url == "" {
		signed := s.signResultURL(task)
		if signed != nil {
			url = strings.TrimSpace(*signed)
		}
	}
	if url != "" {
		resp.Data = []model.ImageData{{URL: url}}
	}

	return resp
}

func (s *generationService) buildVideoResponseFromTask(task *model.GenerationTask) *model.VideoGenerationResponse {
	if task == nil {
		return nil
	}

	resp := &model.VideoGenerationResponse{
		ID:        task.ID.String(),
		Status:    task.Status,
		Progress:  task.Progress,
		CreatedAt: task.CreatedAt.Unix(),
	}

	url := ""
	if task.ResultURL != nil {
		url = strings.TrimSpace(*task.ResultURL)
	}
	if url == "" {
		signed := s.signResultURL(task)
		if signed != nil {
			url = strings.TrimSpace(*signed)
		}
	}
	if url != "" {
		resp.Data = &model.VideoData{URL: url}
	}
	if task.ErrorMessage != nil && strings.TrimSpace(*task.ErrorMessage) != "" {
		msg := strings.TrimSpace(*task.ErrorMessage)
		resp.Error = &msg
	}

	return resp
}

func (s *generationService) loadIdempotentTask(ctx context.Context, userID uuid.UUID, operation string, idempotencyKey string, requestHash string) (*model.GenerationTask, error) {
	if s == nil || s.idempotencyRepo == nil {
		return nil, nil
	}
	if idempotencyKey == "" {
		return nil, nil
	}

	record, err := s.idempotencyRepo.GetByUserOperationKey(ctx, userID, operation, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}

	if record.ExpiresAt != nil && time.Now().After(*record.ExpiresAt) {
		return nil, nil
	}
	if requestHash != "" && record.RequestHash != "" && record.RequestHash != requestHash {
		return nil, ErrIdempotencyConflict
	}

	task, err := s.generationRepo.GetByID(record.ResourceID)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate key value") {
		return true
	}
	if strings.Contains(msg, "unique") && strings.Contains(msg, "constraint") {
		return true
	}
	return false
}

func (s *generationService) tryCreateIdempotencyRecord(ctx context.Context, token *model.Token, operation string, idempotencyKey string, requestHash string, resourceID uuid.UUID, ttl time.Duration) error {
	if s == nil || s.idempotencyRepo == nil {
		return nil
	}
	if token == nil || token.UserID == uuid.Nil || token.ID == uuid.Nil {
		return nil
	}
	if idempotencyKey == "" || requestHash == "" || resourceID == uuid.Nil {
		return nil
	}

	var expiresAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expiresAt = &t
	}

	record := &model.IdempotencyKey{
		UserID:       token.UserID,
		TokenID:      token.ID,
		Operation:    model.NormalizeOperation(operation),
		Key:          idempotencyKey,
		RequestHash:  requestHash,
		ResourceType: "generation_task",
		ResourceID:   resourceID,
		Status:       "completed",
		ExpiresAt:    expiresAt,
	}

	err := s.idempotencyRepo.Create(ctx, record)
	if err != nil && isUniqueViolation(err) {
		return nil
	}
	return err
}

func (s *generationService) isTaskExpired(task *model.GenerationTask) bool {
	if task == nil {
		return false
	}
	if task.Status == model.GenerationStatusExpired {
		return true
	}
	if task.ExpiresAt != nil && time.Now().After(*task.ExpiresAt) {
		return true
	}
	return false
}

func (s *generationService) signResultURL(task *model.GenerationTask) *string {
	if task == nil {
		return nil
	}
	if s.isTaskExpired(task) {
		return nil
	}

	// 閺傜増鏆熼幑顕嗙窗娴兼ê鍘涙担璺ㄦ暏 object_key 閸斻劍鈧胶顒烽崥?
	if task.ResultObjectKey != nil && strings.TrimSpace(*task.ResultObjectKey) != "" && s.objStore != nil {
		if url, err := s.objStore.SignGetURL(strings.TrimSpace(*task.ResultObjectKey), s.signExpireSeconds); err == nil {
			return &url
		}
	}

	// 閸忕厧顔愰弮褎鏆熼幑顕嗙窗閸ョ偤鈧偓閸掓澘鐡ㄦ惔鎾舵畱 result_url閿涘牆褰查懗鍊熺箖閺堢噦绱?
	if task.ResultURL != nil && strings.TrimSpace(*task.ResultURL) != "" {
		u := strings.TrimSpace(*task.ResultURL)
		return &u
	}
	return nil
}

// ResourcePoolRequest ?????
type ResourcePoolRequest struct {
	Provider        string                 `json:"provider"`
	UserID          uuid.UUID              `json:"user_id"`
	ModelName       string                 `json:"model_name"`
	DefaultParams   map[string]interface{} `json:"default_params,omitempty"`
	UserInputs      map[string]interface{} `json:"user_inputs"`
	AdminFixed      map[string]interface{} `json:"admin_fixed,omitempty"`
	AdapterSettings map[string]interface{} `json:"adapter_settings,omitempty"`
}

// ResourcePoolResponse ?????
type ResourcePoolResponse struct {
	Data struct {
		Status          string                 `json:"status"`
		Progress        float64                `json:"progress"`
		Message         string                 `json:"message,omitempty"`
		ResultURL       string                 `json:"result_url,omitempty"`
		ResultObjectKey string                 `json:"result_object_key,omitempty"`
		ResultData      map[string]interface{} `json:"result_data,omitempty"`
		Error           string                 `json:"error,omitempty"`
	} `json:"data"`
}

func buildResourcePoolAdapterSettings(channel *model.Channel) map[string]interface{} {
	if channel == nil {
		return nil
	}

	settings := map[string]interface{}{}
	if channel.BaseURL != "" {
		settings["base_url"] = channel.BaseURL
	}
	// 濞夈劍鍓伴敍姘瑝閼虫垝绱剁粚鍝勭摟缁楋缚瑕嗛敍灞芥儊閸掓瑤绱扮憰鍡欐磰鐠у嫭绨Ч鐘辨櫠閻ㄥ嫮骞嗘晶鍐ㄥ綁闁插繘绮拋銈呪偓纭风礄鐏忋倕鍙鹃弰?Sora2Adapter閿?
	if channel.APIKey != "" {
		settings["api_key"] = channel.APIKey
	}

	if len(settings) == 0 {
		return nil
	}
	return settings
}

func resolveResourcePoolProviderKey(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	if name == "" {
		return "", fmt.Errorf("resource-pool provider is empty")
	}

	known := map[string]struct{}{
		"banana":   {},
		"seedream": {},
		"sora2":    {},
		"jimeng":   {},
	}

	tokens := strings.FieldsFunc(name, func(r rune) bool {
		if r >= 'a' && r <= 'z' {
			return false
		}
		if r >= '0' && r <= '9' {
			return false
		}
		return true
	})

	for _, token := range tokens {
		if _, ok := known[token]; ok {
			return token, nil
		}
	}

	return "", fmt.Errorf("resource-pool provider unsupported: %s", raw)
}

func getResourcePoolConfigString(channel *model.Channel, key string) (string, bool) {
	if channel == nil || channel.Config == nil {
		return "", false
	}
	v, ok := channel.Config[key]
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

func getResourcePoolNestedConfigString(channel *model.Channel, parentKey string, key string) (string, bool) {
	if channel == nil || channel.Config == nil {
		return "", false
	}
	v, ok := channel.Config[parentKey]
	if !ok || v == nil {
		return "", false
	}
	m, ok := v.(map[string]any)
	if !ok || m == nil {
		return "", false
	}
	raw, ok := m[key]
	if !ok || raw == nil {
		return "", false
	}
	s, ok := raw.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

func (s *generationService) resolveResourcePoolURL(channel *model.Channel) (string, error) {
	if url, ok := getResourcePoolConfigString(channel, "resource_pool_url"); ok {
		return strings.TrimRight(url, "/"), nil
	}
	if url, ok := getResourcePoolNestedConfigString(channel, "resource_pool", "url"); ok {
		return strings.TrimRight(url, "/"), nil
	}

	url := strings.TrimRight(strings.TrimSpace(s.resourcePoolURL), "/")
	if url == "" {
		return "", fmt.Errorf("resource-pool url is empty")
	}
	return url, nil
}

func (s *generationService) resolveResourcePoolProviderKeyFromChannel(channel *model.Channel, upstreamModelName string, requestModel string) (string, error) {
	candidates := make([]string, 0, 5)

	if v, ok := getResourcePoolConfigString(channel, "resource_pool_provider"); ok {
		candidates = append(candidates, v)
	}
	if v, ok := getResourcePoolNestedConfigString(channel, "resource_pool", "provider"); ok {
		candidates = append(candidates, v)
	}
	if upstreamModelName != "" {
		candidates = append(candidates, upstreamModelName)
	}
	if requestModel != "" {
		candidates = append(candidates, requestModel)
	}
	if channel != nil && channel.Name != "" {
		candidates = append(candidates, channel.Name)
	}

	for _, candidate := range candidates {
		key, err := resolveResourcePoolProviderKey(candidate)
		if err == nil {
			return key, nil
		}
	}

	return "", fmt.Errorf("resource-pool provider is empty")
}

func (s *generationService) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest, token *model.Token, idempotencyKey string) (*model.ImageGenerationResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	if token == nil {
		return nil, fmt.Errorf("token is nil")
	}

	startTime := time.Now()
	idKey := normalizeIdempotencyKey(idempotencyKey)
	reqHash := hashRequest(req)
	if idKey != "" && reqHash != "" {
		task, err := s.loadIdempotentTask(ctx, token.UserID, model.OperationImagesGenerations, idKey, reqHash)
		if err != nil {
			return nil, err
		}
		if task != nil {
			return s.buildImageResponseFromTask(task), nil
		}
	}

	modelPricing, err := s.modelRepo.GetPricing(req.Model)
	if err != nil {
		modelPricing = &model.ModelPricing{
			ModelID:     req.Model,
			InputPrice:  decimal.NewFromFloat(0.01),
			OutputPrice: decimal.NewFromFloat(0.01),
			PriceUnit:   1,
		}
	}

	estimatedCost := s.calculateImageCost(modelPricing, req)
	if err := s.ensureSufficientBalance(ctx, token, estimatedCost); err != nil {
		return nil, err
	}

	taskID := uuid.New()
	task := &model.GenerationTask{
		ID:       taskID,
		UserID:   token.UserID,
		TokenID:  token.ID,
		Type:     model.GenerationTypeImage,
		Model:    req.Model,
		Provider: "unknown",
		Prompt:   req.Prompt,
		Params: model.JSON{
			"size":            req.Size,
			"aspect_ratio":    req.AspectRatio,
			"resolution":      req.Resolution,
			"response_format": req.ResponseFormat,
			"n":               req.N,
			"urls":            req.URLs,
			"image":           req.Image,
			"seed":            req.Seed,
			"watermark":       req.Watermark,
		},
		Status:   model.GenerationStatusProcessing,
		Progress: 0,
	}
	if err := s.generationRepo.Create(task); err != nil {
		return nil, fmt.Errorf("閸掓稑缂撴禒璇插婢惰精瑙? %w", err)
	}

	_ = s.tryCreateIdempotencyRecord(ctx, token, model.OperationImagesGenerations, idKey, reqHash, taskID, 24*time.Hour)

	if err := s.processImageTask(ctx, req, token, task, modelPricing, startTime, false); err != nil {
		return nil, err
	}

	return s.buildImageResponseFromTask(task), nil
}

func (s *generationService) CreateImageTask(ctx context.Context, req *model.ImageGenerationRequest, token *model.Token, idempotencyKey string) (*model.ImageGenerationResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	if token == nil {
		return nil, fmt.Errorf("token is nil")
	}

	idKey := normalizeIdempotencyKey(idempotencyKey)
	reqHash := hashRequest(req)
	if idKey != "" && reqHash != "" {
		task, err := s.loadIdempotentTask(ctx, token.UserID, model.OperationImagesGenerations, idKey, reqHash)
		if err != nil {
			return nil, err
		}
		if task != nil {
			return s.buildImageResponseFromTask(task), nil
		}
	}

	modelPricing, err := s.modelRepo.GetPricing(req.Model)
	if err != nil {
		modelPricing = &model.ModelPricing{
			ModelID:     req.Model,
			InputPrice:  decimal.NewFromFloat(0.01),
			OutputPrice: decimal.NewFromFloat(0.01),
			PriceUnit:   1,
		}
	}
	estimatedCost := s.calculateImageCost(modelPricing, req)
	if err := s.ensureSufficientBalance(ctx, token, estimatedCost); err != nil {
		return nil, err
	}

	taskID := uuid.New()
	task := &model.GenerationTask{
		ID:       taskID,
		UserID:   token.UserID,
		TokenID:  token.ID,
		Type:     model.GenerationTypeImage,
		Model:    req.Model,
		Provider: "unknown",
		Prompt:   req.Prompt,
		Params: model.JSON{
			"size":            req.Size,
			"aspect_ratio":    req.AspectRatio,
			"resolution":      req.Resolution,
			"response_format": req.ResponseFormat,
			"n":               req.N,
			"urls":            req.URLs,
			"image":           req.Image,
			"seed":            req.Seed,
			"watermark":       req.Watermark,
		},
		Status:   model.GenerationStatusPending,
		Progress: 0,
	}
	if err := s.generationRepo.Create(task); err != nil {
		return nil, fmt.Errorf("閸掓稑缂撴禒璇插婢惰精瑙? %w", err)
	}

	_ = s.tryCreateIdempotencyRecord(ctx, token, model.OperationImagesGenerations, idKey, reqHash, taskID, 24*time.Hour)
	return s.buildImageResponseFromTask(task), nil
}

func (s *generationService) GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest, token *model.Token, idempotencyKey string) (*model.VideoGenerationResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	if token == nil {
		return nil, fmt.Errorf("token is nil")
	}

	idKey := normalizeIdempotencyKey(idempotencyKey)
	reqHash := hashRequest(req)
	if idKey != "" && reqHash != "" {
		task, err := s.loadIdempotentTask(ctx, token.UserID, model.OperationVideosGenerations, idKey, reqHash)
		if err != nil {
			return nil, err
		}
		if task != nil {
			return s.buildVideoResponseFromTask(task), nil
		}
	}

	modelPricing, err := s.modelRepo.GetPricing(req.Model)
	if err != nil {
		modelPricing = &model.ModelPricing{
			ModelID:     req.Model,
			InputPrice:  decimal.NewFromFloat(0.1),
			OutputPrice: decimal.NewFromFloat(0.1),
			PriceUnit:   1,
		}
	}

	estimatedCost := s.calculateVideoCost(modelPricing, req)
	if err := s.ensureSufficientBalance(ctx, token, estimatedCost); err != nil {
		return nil, err
	}

	taskID := uuid.New()
	task := &model.GenerationTask{
		ID:       taskID,
		UserID:   token.UserID,
		TokenID:  token.ID,
		Type:     model.GenerationTypeVideo,
		Model:    req.Model,
		Provider: "unknown",
		Prompt:   req.Prompt,
		Params: model.JSON{
			"aspect_ratio": req.AspectRatio,
			"duration":     req.Duration,
			"size":         req.Size,
			"image_url":    req.ImageURL,
		},
		Status:   model.GenerationStatusPending,
		Progress: 0,
	}
	if err := s.generationRepo.Create(task); err != nil {
		return nil, fmt.Errorf("閸掓稑缂撴禒璇插婢惰精瑙? %w", err)
	}

	_ = s.tryCreateIdempotencyRecord(ctx, token, model.OperationVideosGenerations, idKey, reqHash, taskID, 24*time.Hour)

	// 视频统一走异步任务，立即返回任务状态。
	return s.buildVideoResponseFromTask(task), nil
}

func (s *generationService) ensureSufficientBalance(ctx context.Context, token *model.Token, estimatedCost decimal.Decimal) error {
	if token == nil {
		return fmt.Errorf("token 娑撳秷鍏樻稉铏光敄")
	}
	if estimatedCost.LessThanOrEqual(decimal.Zero) {
		return nil
	}

	// 閻劍鍩涙担娆擃杺閺嶏繝鐛?
	var balance decimal.Decimal
	balanceKnown := false
	if token.User != nil {
		balance = token.User.Balance
		balanceKnown = true
	} else if s != nil && s.userRepo != nil {
		user, err := s.userRepo.GetByID(token.UserID)
		if err != nil {
			return fmt.Errorf("閼惧嘲褰囬悽銊﹀煕婢惰精瑙? %w", err)
		}
		if user != nil {
			balance = user.Balance
			balanceKnown = true
		}
	}

	if !balanceKnown {
		// 閺冪姵纭剁涵顔肩暰娴ｆ瑩顤傞弮璁圭礉娑撳秵瀚ら幋顏庣幢闁灝鍘ら崶鐘辫礋缂傚搫鐨?preload/DB 娑撳瓨妞傞梻顕€顣介懓宀冾嚖閺夆偓
		return nil
	}
	if balance.LessThan(estimatedCost) {
		return &InsufficientFundsError{Needed: estimatedCost, Balance: balance}
	}

	// Token 闁板秹顤傞弽锟犵崣
	if !token.HasQuota(estimatedCost) {
		remaining := decimal.Zero
		if token.RemainQuota != nil {
			remaining = *token.RemainQuota
		}
		return &QuotaExceededError{Needed: estimatedCost, Remaining: remaining}
	}

	return nil
}

func (s *generationService) processImageTask(ctx context.Context, req *model.ImageGenerationRequest, token *model.Token, task *model.GenerationTask, modelPricing *model.ModelPricing, startTime time.Time, allowIncomplete bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil || token == nil || task == nil {
		return fmt.Errorf("invalid request or task")
	}

	fail := func(provider *model.ModelProvider, err error, progress float64) error {
		if err == nil {
			err = errors.New("image generation failed")
		}
		_ = s.failTask(task, startTime, err, progress)
		s.createGenerationLog(
			token,
			provider,
			req.Model,
			model.OperationImagesGenerations,
			task.ID,
			decimal.Zero,
			decimal.Zero,
			int(time.Since(startTime).Milliseconds()),
			model.LogStatusError,
			err.Error(),
		)
		return err
	}

	if s.cascadeController == nil {
		initErr := errors.New("cascade controller not initialized")
		return fail(nil, initErr, 0)
	}

	rateCtx := scheduler.WithRateLimitUsage(ctx, buildImageRateLimitUsage(req))

	executor := func(execCtx context.Context, provider *model.ModelProvider) (interface{}, error) {
		if execCtx == nil {
			execCtx = context.Background()
		}
		if provider == nil || provider.Channel == nil || provider.Channel.Name == "" {
			return nil, fmt.Errorf("provider channel is empty")
		}

		resourcePoolURL, err := s.resolveResourcePoolURL(provider.Channel)
		if err != nil {
			return nil, err
		}
		upstreamModelName := provider.UpstreamModelName
		if upstreamModelName == "" {
			upstreamModelName = req.Model
		}
		resourcePoolProvider, err := s.resolveResourcePoolProviderKeyFromChannel(provider.Channel, upstreamModelName, req.Model)
		if err != nil {
			return nil, err
		}

		userInputs := map[string]interface{}{
			"prompt":       req.Prompt,
			"aspect_ratio": req.AspectRatio,
			"resolution":   req.Resolution,
			"seed":         req.Seed,
			"watermark":    req.Watermark,
		}
		if len(req.URLs) > 0 {
			userInputs["urls"] = req.URLs
		}
		if req.Image != "" {
			userInputs["image"] = req.Image
			if len(req.URLs) == 0 {
				userInputs["urls"] = []string{req.Image}
			}
		}
		imageSize := req.Size
		if imageSize == "" {
			imageSize = req.Resolution
		}
		if imageSize != "" {
			userInputs["image_size"] = imageSize
		}

		resourceReq := &ResourcePoolRequest{
			Provider:        resourcePoolProvider,
			UserID:          token.UserID,
			ModelName:       upstreamModelName,
			DefaultParams:   map[string]interface{}{},
			UserInputs:      userInputs,
			AdminFixed:      map[string]interface{}{},
			AdapterSettings: buildResourcePoolAdapterSettings(provider.Channel),
		}

		// CommitGuard: 确保异步任务固定到指定的 provider/instance。
		if s.commitGuard != nil && task != nil {
			var instanceID uint
			if provider.SelectedInstance != nil {
				instanceID = provider.SelectedInstance.ID
			}
			if _, err := s.commitGuard.EnsureJobCommit(execCtx, task.ID.String(), provider.ID, instanceID, 24*time.Hour); err != nil {
				return nil, err
			}
		}
		return s.callResourcePool(execCtx, resourcePoolURL, resourceReq)
	}

	var cascadeResult *scheduler.CascadeResult
	var err error
	if s.commitGuard != nil && s.providerRepo != nil && s.cascadeController != nil && task != nil {
		if commit, _ := s.commitGuard.GetJobCommit(rateCtx, task.ID.String()); commit != nil {
			lockedProvider, pErr := s.providerRepo.GetByID(rateCtx, commit.ProviderID)
			if pErr != nil {
				return fail(nil, pErr, 0)
			}
			if lockedProvider == nil {
				return fail(nil, fmt.Errorf("瀹告煡鏀ｇ€规碍绨径缈犵瑝鐎涙ê婀? %d", commit.ProviderID), 0)
			}
			lockedProvider.SelectedInstance = &model.ProviderInstance{ID: commit.InstanceID}
			cascadeResult, err = s.cascadeController.ExecuteOnProvider(rateCtx, model.OperationImagesGenerations, req.Model, lockedProvider, executor)
		} else {
			cascadeResult, err = s.cascadeController.ExecuteWithStrategy(rateCtx, model.OperationImagesGenerations, req.Model, scheduler.StrategyCostFirst, executor)
		}
	} else {
		cascadeResult, err = s.cascadeController.ExecuteWithStrategy(rateCtx, model.OperationImagesGenerations, req.Model, scheduler.StrategyCostFirst, executor)
	}
	if err != nil {
		return fail(nil, err, 0)
	}

	providerName, resourceResp, err := s.unwrapResourcePoolResult(cascadeResult)
	if err != nil {
		return fail(cascadeResult.Provider, err, 0)
	}
	task.Provider = providerName

	status := strings.ToLower(strings.TrimSpace(resourceResp.Data.Status))
	resultURL := strings.TrimSpace(resourceResp.Data.ResultURL)
	resultObjectKey := strings.TrimSpace(resourceResp.Data.ResultObjectKey)
	completed := status == "completed" && (resultURL != "" || resultObjectKey != "")
	if !completed {
		if allowIncomplete && status != "failed" {
			progress := resourceResp.Data.Progress
			if progress < 0 {
				progress = 0
			}
			if progress > 1 {
				progress = 1
			}
			task.Status = model.GenerationStatusPending
			task.Progress = progress
			if resourceResp.Data.ResultData != nil {
				task.ResultData = resourceResp.Data.ResultData
			}
			if err := s.generationRepo.Update(task); err != nil {
				return fmt.Errorf("鏇存柊浠诲姟澶辫触: %w", err)
			}
			return nil
		}

		errMsg := resourceResp.Data.Error
		if errMsg == "" {
			errMsg = "鍥剧墖鐢熸垚澶辫触"
		}
		return fail(cascadeResult.Provider, errors.New(errMsg), resourceResp.Data.Progress)
	}

	cost := s.calculateImageCost(modelPricing, req)
	if cascadeResult != nil && cascadeResult.Provider != nil {
		if charged, ok := s.resolveCharge(ctx, cascadeResult.Provider.ID, model.OperationImagesGenerations, "image", int64(req.N)); ok {
			cost = charged
		}
	}
	upstreamCost := decimal.Zero
	if cascadeResult != nil && cascadeResult.Provider != nil {
		if charged, ok := s.resolveUpstreamCost(ctx, cascadeResult.Provider.ID, model.OperationImagesGenerations, "image", int64(req.N)); ok {
			upstreamCost = charged
		}
	}
	resultURL, resultObjectKey = s.tryStoreResultObject(rateCtx, task, resultURL, resultObjectKey)
	now := time.Now()

	task.Status = model.GenerationStatusCompleted
	task.Progress = 1.0
	task.ResultURL = &resultURL
	if resultObjectKey != "" {
		task.ResultObjectKey = &resultObjectKey
	}
	if resourceResp.Data.ResultData != nil {
		task.ResultData = resourceResp.Data.ResultData
	}
	task.Cost = cost
	task.Duration = int(time.Since(startTime).Milliseconds())
	task.CompletedAt = &now
	task.ErrorMessage = nil

	if err := s.generationRepo.Update(task); err != nil {
		return fmt.Errorf("閺囧瓨鏌婃禒璇插婢惰精瑙? %w", err)
	}

	_ = s.userRepo.DeductBalance(token.UserID, cost)
	if token.RemainQuota != nil {
		_ = s.tokenService.DeductQuota(token.ID, cost)
	}
	s.createGenerationLog(token, cascadeResult.Provider, req.Model, model.OperationImagesGenerations, task.ID, cost, upstreamCost, task.Duration, model.LogStatusSuccess, "")

	return nil
}

func (s *generationService) processVideoTask(ctx context.Context, req *model.VideoGenerationRequest, token *model.Token, task *model.GenerationTask, modelPricing *model.ModelPricing, startTime time.Time, allowIncomplete bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil || token == nil || task == nil {
		return fmt.Errorf("invalid request or task")
	}

	fail := func(provider *model.ModelProvider, err error, progress float64) error {
		if err == nil {
			err = errors.New("video generation failed")
		}
		_ = s.failTask(task, startTime, err, progress)
		s.createGenerationLog(
			token,
			provider,
			req.Model,
			model.OperationVideosGenerations,
			task.ID,
			decimal.Zero,
			decimal.Zero,
			int(time.Since(startTime).Milliseconds()),
			model.LogStatusError,
			err.Error(),
		)
		return err
	}

	if s.cascadeController == nil {
		initErr := errors.New("cascade controller not initialized")
		return fail(nil, initErr, 0)
	}

	rateCtx := scheduler.WithRateLimitUsage(ctx, buildVideoRateLimitUsage(req))

	executor := func(execCtx context.Context, provider *model.ModelProvider) (interface{}, error) {
		if execCtx == nil {
			execCtx = context.Background()
		}
		if provider == nil || provider.Channel == nil || provider.Channel.Name == "" {
			return nil, fmt.Errorf("provider channel is empty")
		}

		resourcePoolURL, err := s.resolveResourcePoolURL(provider.Channel)
		if err != nil {
			return nil, err
		}
		upstreamModelName := provider.UpstreamModelName
		if upstreamModelName == "" {
			upstreamModelName = req.Model
		}
		resourcePoolProvider, err := s.resolveResourcePoolProviderKeyFromChannel(provider.Channel, upstreamModelName, req.Model)
		if err != nil {
			return nil, err
		}

		userInputs := map[string]interface{}{
			"prompt":       req.Prompt,
			"aspect_ratio": req.AspectRatio,
			"duration":     req.Duration,
			"size":         req.Size,
		}
		if req.ImageURL != "" {
			userInputs["image"] = req.ImageURL
		}

		resourceReq := &ResourcePoolRequest{
			Provider:        resourcePoolProvider,
			UserID:          token.UserID,
			ModelName:       upstreamModelName,
			DefaultParams:   map[string]interface{}{},
			UserInputs:      userInputs,
			AdminFixed:      map[string]interface{}{},
			AdapterSettings: buildResourcePoolAdapterSettings(provider.Channel),
		}

		// 鐚?CommitGuard閿涘湞ob 閹绘劒姘﹂悙褰掓，閹貉嶇礆
		if s.commitGuard != nil && task != nil {
			var instanceID uint
			if provider.SelectedInstance != nil {
				instanceID = provider.SelectedInstance.ID
			}
			if _, err := s.commitGuard.EnsureJobCommit(execCtx, task.ID.String(), provider.ID, instanceID, 24*time.Hour); err != nil {
				return nil, err
			}
		}
		return s.callResourcePool(execCtx, resourcePoolURL, resourceReq)
	}

	var cascadeResult *scheduler.CascadeResult
	var err error
	if s.commitGuard != nil && s.providerRepo != nil && s.cascadeController != nil && task != nil {
		if commit, _ := s.commitGuard.GetJobCommit(rateCtx, task.ID.String()); commit != nil {
			lockedProvider, pErr := s.providerRepo.GetByID(rateCtx, commit.ProviderID)
			if pErr != nil {
				return fail(nil, pErr, 0)
			}
			if lockedProvider == nil {
				return fail(nil, fmt.Errorf("瀹告煡鏀ｇ€规碍绨径缈犵瑝鐎涙ê婀? %d", commit.ProviderID), 0)
			}
			lockedProvider.SelectedInstance = &model.ProviderInstance{ID: commit.InstanceID}
			cascadeResult, err = s.cascadeController.ExecuteOnProvider(rateCtx, model.OperationVideosGenerations, req.Model, lockedProvider, executor)
		} else {
			cascadeResult, err = s.cascadeController.ExecuteWithStrategy(rateCtx, model.OperationVideosGenerations, req.Model, scheduler.StrategyCostFirst, executor)
		}
	} else {
		cascadeResult, err = s.cascadeController.ExecuteWithStrategy(rateCtx, model.OperationVideosGenerations, req.Model, scheduler.StrategyCostFirst, executor)
	}
	if err != nil {
		return fail(nil, err, 0)
	}

	providerName, resourceResp, err := s.unwrapResourcePoolResult(cascadeResult)
	if err != nil {
		return fail(cascadeResult.Provider, err, 0)
	}
	task.Provider = providerName

	status := strings.ToLower(strings.TrimSpace(resourceResp.Data.Status))
	resultURL := strings.TrimSpace(resourceResp.Data.ResultURL)
	resultObjectKey := strings.TrimSpace(resourceResp.Data.ResultObjectKey)
	completed := status == "completed" && (resultURL != "" || resultObjectKey != "")
	if !completed {
		if allowIncomplete && status != "failed" {
			progress := resourceResp.Data.Progress
			if progress < 0 {
				progress = 0
			}
			if progress > 1 {
				progress = 1
			}
			task.Status = model.GenerationStatusPending
			task.Progress = progress
			if resourceResp.Data.ResultData != nil {
				task.ResultData = resourceResp.Data.ResultData
			}
			if err := s.generationRepo.Update(task); err != nil {
				return fmt.Errorf("鏇存柊浠诲姟澶辫触: %w", err)
			}
			return nil
		}

		errMsg := resourceResp.Data.Error
		if errMsg == "" {
			errMsg = "瑙嗛鐢熸垚澶辫触"
		}
		return fail(cascadeResult.Provider, errors.New(errMsg), resourceResp.Data.Progress)
	}

	cost := s.calculateVideoCost(modelPricing, req)
	if cascadeResult != nil && cascadeResult.Provider != nil {
		if charged, ok := s.resolveCharge(ctx, cascadeResult.Provider.ID, model.OperationVideosGenerations, "video_second", int64(req.Duration)); ok {
			cost = charged
		}
	}
	upstreamCost := decimal.Zero
	if cascadeResult != nil && cascadeResult.Provider != nil {
		if charged, ok := s.resolveUpstreamCost(ctx, cascadeResult.Provider.ID, model.OperationVideosGenerations, "video_second", int64(req.Duration)); ok {
			upstreamCost = charged
		}
	}
	resultURL, resultObjectKey = s.tryStoreResultObject(rateCtx, task, resultURL, resultObjectKey)
	now := time.Now()

	task.Status = model.GenerationStatusCompleted
	task.Progress = 1.0
	task.ResultURL = &resultURL
	if resultObjectKey != "" {
		task.ResultObjectKey = &resultObjectKey
	}
	if resourceResp.Data.ResultData != nil {
		task.ResultData = resourceResp.Data.ResultData
	}
	task.Cost = cost
	task.Duration = int(time.Since(startTime).Milliseconds())
	task.CompletedAt = &now
	task.ErrorMessage = nil

	if err := s.generationRepo.Update(task); err != nil {
		return fmt.Errorf("閺囧瓨鏌婃禒璇插婢惰精瑙? %w", err)
	}

	_ = s.userRepo.DeductBalance(token.UserID, cost)
	if token.RemainQuota != nil {
		_ = s.tokenService.DeductQuota(token.ID, cost)
	}

	s.createGenerationLog(token, cascadeResult.Provider, req.Model, model.OperationVideosGenerations, task.ID, cost, upstreamCost, task.Duration, model.LogStatusSuccess, "")
	return nil
}

func (s *generationService) GetTaskStatus(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) (*model.GenerationTaskResponse, error) {
	_ = ctx
	task, err := s.generationRepo.GetByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGenerationTaskNotFound
		}
		return nil, fmt.Errorf("閼惧嘲褰囨禒璇插婢惰精瑙? %w", err)
	}
	if task.UserID != userID {
		return nil, ErrGenerationTaskForbidden
	}

	resp := task.ToResponse()
	if s.isTaskExpired(task) {
		resp.Status = model.GenerationStatusExpired
		resp.ResultURL = nil
		if resp.ErrorMessage == nil {
			msg := "result expired"
			resp.ErrorMessage = &msg
		}
		return resp, nil
	}

	resp.ResultURL = s.signResultURL(task)
	return resp, nil
}

func (s *generationService) ListUserTasks(ctx context.Context, userID uuid.UUID, taskType model.GenerationType, page, pageSize int) ([]model.GenerationTaskResponse, int64, error) {
	_ = ctx
	tasks, total, err := s.generationRepo.GetByUserID(userID, taskType, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]model.GenerationTaskResponse, 0, len(tasks))
	for i := range tasks {
		task := tasks[i]
		resp := task.ToResponse()
		if s.isTaskExpired(&task) {
			resp.Status = model.GenerationStatusExpired
			resp.ResultURL = nil
			if resp.ErrorMessage == nil {
				msg := "result expired"
				resp.ErrorMessage = &msg
			}
		} else {
			resp.ResultURL = s.signResultURL(&task)
		}
		responses = append(responses, *resp)
	}

	return responses, total, nil
}

func (s *generationService) callResourcePool(ctx context.Context, resourcePoolURL string, req *ResourcePoolRequest) (*ResourcePoolResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	resourcePoolURL = strings.TrimRight(strings.TrimSpace(resourcePoolURL), "/")
	if resourcePoolURL == "" {
		return nil, fmt.Errorf("resource-pool url is empty")
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("鎼村繐鍨崠鏍嚞濮瑰倸銇戠拹? %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, resourcePoolURL+"/v1/generate", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("閸掓稑缂撶拠閿嬬湴婢惰精瑙? %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("鐠囬攱鐪扮挧鍕爱濮圭姴銇戠拹? %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("鐠у嫭绨Ч鐘虹箲閸ョ偤鏁婄拠? %d - %s", resp.StatusCode, string(body))
	}

	var result ResourcePoolResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("鐟欙絾鐎介崫宥呯安婢惰精瑙? %w", err)
	}

	return &result, nil
}

func (s *generationService) unwrapResourcePoolResult(result *scheduler.CascadeResult) (providerName string, resourceResp *ResourcePoolResponse, err error) {
	if result == nil || result.Provider == nil || result.Provider.Channel == nil || result.Provider.Channel.Name == "" {
		return "", nil, errors.New("resource pool provider missing")
	}

	providerName = result.Provider.Channel.Name
	resourceResp, ok := result.Response.(*ResourcePoolResponse)
	if !ok || resourceResp == nil {
		return providerName, nil, errors.New("resource pool response invalid")
	}

	return providerName, resourceResp, nil
}

func (s *generationService) failTask(task *model.GenerationTask, startTime time.Time, err error, progress float64) error {
	if task == nil {
		return nil
	}

	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	now := time.Now()
	task.Status = model.GenerationStatusFailed
	task.Progress = progress
	if errMsg != "" {
		task.ErrorMessage = &errMsg
	}
	task.Duration = int(time.Since(startTime).Milliseconds())
	task.CompletedAt = &now
	return s.generationRepo.Update(task)
}

func (s *generationService) calculateImageCost(pricing *model.ModelPricing, req *model.ImageGenerationRequest) decimal.Decimal {
	if pricing == nil {
		return decimal.Zero
	}

	baseCost := pricing.InputPrice
	resolutionMultiplier := decimal.NewFromFloat(1.0)
	switch req.Resolution {
	case "2K":
		resolutionMultiplier = decimal.NewFromFloat(1.5)
	case "4K":
		resolutionMultiplier = decimal.NewFromFloat(2.0)
	}

	n := req.N
	if n <= 0 {
		n = 1
	}

	return baseCost.Mul(resolutionMultiplier).Mul(decimal.NewFromInt(int64(n)))
}

func (s *generationService) calculateVideoCost(pricing *model.ModelPricing, req *model.VideoGenerationRequest) decimal.Decimal {
	if pricing == nil {
		return decimal.Zero
	}

	baseCost := pricing.InputPrice

	duration := req.Duration
	if duration <= 0 {
		duration = 10
	}
	durationMultiplier := decimal.NewFromFloat(float64(duration) / 10.0)

	sizeMultiplier := decimal.NewFromFloat(1.0)
	if req.Size == "large" {
		sizeMultiplier = decimal.NewFromFloat(1.5)
	}

	return baseCost.Mul(durationMultiplier).Mul(sizeMultiplier)
}

func buildImageRateLimitUsage(req *model.ImageGenerationRequest) map[string]int64 {
	if req == nil {
		return nil
	}

	n := req.N
	if n <= 0 {
		n = 1
	}

	return map[string]int64{
		"request": 1,
		"image":   int64(n),
		"images":  int64(n),
	}
}

func buildVideoRateLimitUsage(req *model.VideoGenerationRequest) map[string]int64 {
	if req == nil {
		return nil
	}

	usage := map[string]int64{
		"request": 1,
	}

	if req.Duration > 0 {
		usage["video_second"] = int64(req.Duration)
		usage["video_seconds"] = int64(req.Duration)
		usage["second"] = int64(req.Duration)
		usage["seconds"] = int64(req.Duration)
	}

	return usage
}

func (s *generationService) tryStoreResultObject(ctx context.Context, task *model.GenerationTask, resultURL string, resultObjectKey string) (string, string) {
	if ctx == nil {
		ctx = context.Background()
	}

	resultURL = strings.TrimSpace(resultURL)
	resultObjectKey = strings.TrimSpace(resultObjectKey)

	if task == nil || resultObjectKey != "" || s == nil || s.objStore == nil || resultURL == "" {
		return resultURL, resultObjectKey
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resultURL, nil)
	if err != nil {
		return resultURL, resultObjectKey
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return resultURL, resultObjectKey
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resultURL, resultObjectKey
	}

	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	objectKey := fmt.Sprintf("generations/%s/%d%s", task.ID.String(), time.Now().UnixNano(), guessResultExt(resultURL, contentType))

	opts := storage.PutObjectOptions{ContentType: contentType}
	if resp.ContentLength > 0 {
		opts.ContentLength = resp.ContentLength
	}

	if err := s.objStore.PutObject(objectKey, resp.Body, opts); err != nil {
		return resultURL, resultObjectKey
	}

	return resultURL, objectKey
}

func guessResultExt(resultURL string, contentType string) string {
	trimmedURL := strings.TrimSpace(resultURL)
	if trimmedURL != "" {
		if idx := strings.Index(trimmedURL, "?"); idx >= 0 {
			trimmedURL = trimmedURL[:idx]
		}
		if ext := strings.ToLower(path.Ext(trimmedURL)); ext != "" && len(ext) <= 6 {
			return ext
		}
	}

	ct := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}

	switch ct {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	default:
		return ""
	}
}
