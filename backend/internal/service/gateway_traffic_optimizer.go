package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/model"
	"nexus-api/internal/service/scheduler"
)

const (
	gatewayDefaultResponseCacheTTL = 45 * time.Second
	gatewayEmbeddingCacheTTL       = 5 * time.Minute
	gatewayIdempotencyTTL          = 24 * time.Hour
	gatewayCacheVersionTTL         = 24 * time.Hour
)

type gatewayRequestControl struct {
	Operation      string
	ModelID        string
	IdempotencyKey string
	EnableCache    bool
	BypassCache    bool
	DisableDedup   bool
}

type gatewayResponseCachePayload struct {
	Response json.RawMessage `json:"response"`
	CachedAt int64           `json:"cached_at"`
}

type gatewayIdempotencySnapshot struct {
	RequestHash string          `json:"request_hash"`
	Response    json.RawMessage `json:"response"`
	UpdatedAt   int64           `json:"updated_at"`
}

func (s *gatewayServiceV2) applyGatewayTrafficHints(ctx context.Context, operation string, modelID string, hints scheduler.TrafficHints) context.Context {
	ctx = scheduler.WithTrafficHints(ctx, hints)
	requestID := hints.IdempotencyKey
	if strings.TrimSpace(requestID) == "" {
		requestID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return scheduler.WithDispatchRequestID(ctx, requestID+"|"+model.NormalizeOperation(operation)+"|"+strings.TrimSpace(modelID))
}

func (s *gatewayServiceV2) executeWithTrafficOptimization[T any](
	ctx context.Context,
	token *model.Token,
	request any,
	control gatewayRequestControl,
	executor func(execCtx context.Context) (*T, error),
) (*T, error) {
	if executor == nil {
		return nil, fmt.Errorf("executor 不能为空")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	operation := model.NormalizeOperation(control.Operation)
	modelID := strings.TrimSpace(control.ModelID)
	idempotencyKey := normalizeIdempotencyKey(control.IdempotencyKey)

	requestHash, err := buildGatewayRequestHash(operation, modelID, request)
	if err != nil {
		return executor(ctx)
	}

	enableCache := control.EnableCache
	if operation == model.OperationEmbeddings && !control.BypassCache {
		enableCache = true
	}

	// 幂等命中优先，确保相同 key 的请求可稳定返回相同响应
	if idempotencyKey != "" {
		hit, resp, conflict := loadGatewayIdempotencySnapshot[T](ctx, token, operation, idempotencyKey, requestHash)
		if conflict {
			return nil, ErrIdempotencyConflict
		}
		if hit {
			return resp, nil
		}
	}

	cacheKey := ""
	if enableCache && !control.BypassCache {
		version := readGatewayCacheVersion(ctx, operation, modelID)
		cacheKey = gatewayResponseCacheKey(operation, modelID, requestHash, version)
		if hit, cached := loadGatewayResponseCache[T](ctx, cacheKey); hit {
			return cached, nil
		}
	}

	doExecute := func() (interface{}, error) {
		resp, execErr := executor(ctx)
		if execErr != nil || resp == nil {
			return resp, execErr
		}

		raw, marshalErr := json.Marshal(resp)
		if marshalErr == nil && len(raw) > 0 {
			if cacheKey != "" {
				storeGatewayResponseCache(ctx, cacheKey, raw, responseCacheTTLByOperation(operation))
			}
			if idempotencyKey != "" {
				storeGatewayIdempotencySnapshot(ctx, token, operation, idempotencyKey, requestHash, raw)
			}
		}

		return resp, nil
	}

	if control.DisableDedup {
		value, execErr := doExecute()
		if execErr != nil {
			return nil, execErr
		}
		if value == nil {
			return nil, nil
		}
		resp, ok := value.(*T)
		if !ok {
			return nil, fmt.Errorf("请求执行结果类型不匹配")
		}
		return resp, nil
	}

	dedupKey := "gateway:dedup:" + operation + ":" + requestHash
	value, execErr, _ := s.requestDeduper.Do(dedupKey, doExecute)
	if execErr != nil {
		return nil, execErr
	}
	if value == nil {
		return nil, nil
	}

	resp, ok := value.(*T)
	if !ok {
		return nil, fmt.Errorf("请求去重结果类型不匹配")
	}
	return resp, nil
}

func buildGatewayRequestHash(operation string, modelID string, request any) (string, error) {
	payload := map[string]any{
		"operation": model.NormalizeOperation(operation),
		"model_id":  strings.TrimSpace(modelID),
		"request":   request,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum[:]), nil
}

func responseCacheTTLByOperation(operation string) time.Duration {
	switch model.NormalizeOperation(operation) {
	case model.OperationEmbeddings:
		return gatewayEmbeddingCacheTTL
	default:
		return gatewayDefaultResponseCacheTTL
	}
}

func gatewayCacheVersionKey(operation string, modelID string) string {
	return "gateway:cache:version:" + model.NormalizeOperation(operation) + ":" + strings.TrimSpace(modelID)
}

func gatewayResponseCacheKey(operation string, modelID string, requestHash string, version int64) string {
	return fmt.Sprintf(
		"gateway:response:%s:%s:v%d:%s",
		model.NormalizeOperation(operation),
		strings.TrimSpace(modelID),
		version,
		requestHash,
	)
}

func gatewayIdempotencyStorageKey(token *model.Token, operation string, key string) string {
	if token == nil {
		return ""
	}
	return fmt.Sprintf("gateway:idempotency:%s:%s:%s", token.ID.String(), model.NormalizeOperation(operation), key)
}

func readGatewayCacheVersion(ctx context.Context, operation string, modelID string) int64 {
	client := cache.GetClient()
	if client == nil {
		return 1
	}
	key := gatewayCacheVersionKey(operation, modelID)
	raw, err := client.Get(ctx, key).Result()
	if err != nil {
		_, _ = client.SetNX(ctx, key, "1", gatewayCacheVersionTTL).Result()
		return 1
	}
	version, parseErr := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if parseErr != nil || version <= 0 {
		return 1
	}
	return version
}

func loadGatewayResponseCache[T any](ctx context.Context, cacheKey string) (bool, *T) {
	client := cache.GetClient()
	if client == nil || strings.TrimSpace(cacheKey) == "" {
		return false, nil
	}
	raw, err := client.Get(ctx, cacheKey).Bytes()
	if err != nil || len(raw) == 0 {
		return false, nil
	}

	var payload gatewayResponseCachePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false, nil
	}
	if len(payload.Response) == 0 {
		return false, nil
	}

	var decoded T
	if err := json.Unmarshal(payload.Response, &decoded); err != nil {
		return false, nil
	}
	return true, &decoded
}

func storeGatewayResponseCache(ctx context.Context, cacheKey string, responseRaw []byte, ttl time.Duration) {
	client := cache.GetClient()
	if client == nil || strings.TrimSpace(cacheKey) == "" || len(responseRaw) == 0 {
		return
	}
	if ttl <= 0 {
		ttl = gatewayDefaultResponseCacheTTL
	}

	payload := gatewayResponseCachePayload{
		Response: responseRaw,
		CachedAt: time.Now().Unix(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = client.Set(ctx, cacheKey, raw, ttl).Err()
}

func loadGatewayIdempotencySnapshot[T any](ctx context.Context, token *model.Token, operation string, idempotencyKey string, requestHash string) (bool, *T, bool) {
	idempotencyKey = normalizeIdempotencyKey(idempotencyKey)
	if idempotencyKey == "" {
		return false, nil, false
	}

	client := cache.GetClient()
	if client == nil {
		return false, nil, false
	}

	storageKey := gatewayIdempotencyStorageKey(token, operation, idempotencyKey)
	if storageKey == "" {
		return false, nil, false
	}

	raw, err := client.Get(ctx, storageKey).Bytes()
	if err != nil || len(raw) == 0 {
		return false, nil, false
	}

	var snapshot gatewayIdempotencySnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return false, nil, false
	}

	if snapshot.RequestHash != "" && requestHash != "" && snapshot.RequestHash != requestHash {
		return false, nil, true
	}
	if len(snapshot.Response) == 0 {
		return false, nil, false
	}

	var decoded T
	if err := json.Unmarshal(snapshot.Response, &decoded); err != nil {
		return false, nil, false
	}
	return true, &decoded, false
}

func storeGatewayIdempotencySnapshot(ctx context.Context, token *model.Token, operation string, idempotencyKey string, requestHash string, responseRaw []byte) {
	idempotencyKey = normalizeIdempotencyKey(idempotencyKey)
	if idempotencyKey == "" || len(responseRaw) == 0 {
		return
	}

	client := cache.GetClient()
	if client == nil {
		return
	}

	storageKey := gatewayIdempotencyStorageKey(token, operation, idempotencyKey)
	if storageKey == "" {
		return
	}

	payload := gatewayIdempotencySnapshot{
		RequestHash: requestHash,
		Response:    responseRaw,
		UpdatedAt:   time.Now().Unix(),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = client.Set(ctx, storageKey, raw, gatewayIdempotencyTTL).Err()
}

// InvalidateGatewayResponseCache 提升缓存版本，实现主动失效。
func InvalidateGatewayResponseCache(operation string, modelID string) {
	client := cache.GetClient()
	if client == nil {
		return
	}

	key := gatewayCacheVersionKey(operation, modelID)
	_, _ = client.Incr(cache.GetContext(), key).Result()
	_ = client.Expire(cache.GetContext(), key, gatewayCacheVersionTTL).Err()
}
