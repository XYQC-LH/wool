package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/observability"
	"nexus-api/internal/repository"
	"nexus-api/internal/service/scheduler"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AudioService 音频能力服务（转写/翻译/TTS）
type AudioService interface {
	Transcribe(ctx context.Context, input *AudioMultipartInput, token *model.Token) (*AudioProxyResponse, error)
	Translate(ctx context.Context, input *AudioMultipartInput, token *model.Token) (*AudioProxyResponse, error)
	Speech(ctx context.Context, req *model.AudioSpeechRequest, token *model.Token, writer http.ResponseWriter) error
}

// AudioMultipartInput multipart/form-data 输入（用于转写/翻译）
type AudioMultipartInput struct {
	Model    string
	FilePath string
	FileName string
	Fields   map[string][]string // 原始表单字段（不含 file），允许多值
}

// AudioProxyResponse 上游透传响应
type AudioProxyResponse struct {
	ContentType string
	Body        []byte
}

type audioService struct {
	userRepo          repository.UserRepository
	tokenService      TokenService
	modelRepo         repository.ModelRepository
	pricingRuleRepo   repository.ProviderPricingRuleRepository
	logRepo           repository.LogRepository
	instanceRepo      repository.ProviderInstanceRepository
	healthTracker     scheduler.HealthTracker
	cascadeController scheduler.CascadeController
	httpClient        *http.Client
}

func NewAudioService(
	userRepo repository.UserRepository,
	tokenService TokenService,
	modelRepo repository.ModelRepository,
	pricingRuleRepo repository.ProviderPricingRuleRepository,
	logRepo repository.LogRepository,
	instanceRepo repository.ProviderInstanceRepository,
	healthTracker scheduler.HealthTracker,
	cascadeController scheduler.CascadeController,
) AudioService {
	return &audioService{
		userRepo:          userRepo,
		tokenService:      tokenService,
		modelRepo:         modelRepo,
		pricingRuleRepo:   pricingRuleRepo,
		logRepo:           logRepo,
		instanceRepo:      instanceRepo,
		healthTracker:     healthTracker,
		cascadeController: cascadeController,
		httpClient:        observability.NewHTTPClient(10 * time.Minute),
	}
}

func (s *audioService) resolvePricingRuleCosts(ctx context.Context, providerID uint, operation string) (downstreamCost, upstreamCost decimal.Decimal, ok bool) {
	if s == nil || s.pricingRuleRepo == nil || providerID == 0 {
		return decimal.Zero, decimal.Zero, false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	rule, err := s.pricingRuleRepo.GetByProviderOperationUnit(ctx, providerID, operation, "request")
	if err != nil || rule == nil || !rule.Enabled {
		return decimal.Zero, decimal.Zero, false
	}

	return rule.PricePerUnit, rule.CostPerUnit, true
}

func (s *audioService) Transcribe(ctx context.Context, input *AudioMultipartInput, token *model.Token) (*AudioProxyResponse, error) {
	return s.executeMultipart(ctx, model.OperationAudioTranscriptions, "/v1/audio/transcriptions", input, token)
}

func (s *audioService) Translate(ctx context.Context, input *AudioMultipartInput, token *model.Token) (*AudioProxyResponse, error) {
	return s.executeMultipart(ctx, model.OperationAudioTranslations, "/v1/audio/translations", input, token)
}

func (s *audioService) executeMultipart(ctx context.Context, operation, endpoint string, input *AudioMultipartInput, token *model.Token) (*AudioProxyResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if input == nil {
		return nil, errors.New("请求不能为空")
	}
	if token == nil {
		return nil, errors.New("token 不能为空")
	}
	if input.Model == "" {
		return nil, errors.New("缺少必需参数: model")
	}
	if input.FilePath == "" || input.FileName == "" {
		return nil, errors.New("缺少必需参数: file")
	}

	startTime := time.Now()

	estimatedCost := decimal.Zero
	if pricing, err := s.modelRepo.GetPricing(input.Model); err == nil && pricing != nil {
		estimatedCost = pricing.InputPrice
	}

	if estimatedCost.GreaterThan(decimal.Zero) {
		if token.User != nil && token.User.Balance.LessThan(estimatedCost) {
			return nil, &InsufficientFundsError{Needed: estimatedCost, Balance: token.User.Balance}
		}
		if token.RemainQuota != nil && !token.UnlimitedQuota && token.RemainQuota.LessThan(estimatedCost) {
			return nil, &QuotaExceededError{Needed: estimatedCost, Remaining: *token.RemainQuota}
		}
	}

	executor := func(execCtx context.Context, provider *model.ModelProvider) (interface{}, error) {
		if execCtx == nil {
			execCtx = context.Background()
		}
		if provider == nil || provider.Channel == nil {
			return nil, fmt.Errorf("源头缺少 channel 信息")
		}

		upstreamModel := provider.UpstreamModelName
		if upstreamModel == "" {
			upstreamModel = input.Model
		}

		pr, pw := io.Pipe()
		mw := multipart.NewWriter(pw)

		go func() {
			defer func() {
				_ = mw.Close()
				_ = pw.Close()
			}()

			// 原样转发表单字段，但覆盖 model
			for k, vals := range input.Fields {
				if k == "model" || k == "file" {
					continue
				}
				for _, v := range vals {
					_ = mw.WriteField(k, v)
				}
			}
			_ = mw.WriteField("model", upstreamModel)

			part, err := mw.CreateFormFile("file", input.FileName)
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}

			f, err := os.Open(input.FilePath)
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			defer f.Close()

			if _, err := io.Copy(part, f); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
		}()

		url := provider.Channel.BaseURL + endpoint
		upstreamReq, err := http.NewRequestWithContext(execCtx, http.MethodPost, url, pr)
		if err != nil {
			return nil, err
		}
		upstreamReq.Header.Set("Content-Type", mw.FormDataContentType())
		upstreamReq.Header.Set("Authorization", "Bearer "+provider.Channel.APIKey)

		resp, err := s.httpClient.Do(upstreamReq)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, readErr
		}

		if resp.StatusCode != http.StatusOK {
			return nil, NewUpstreamHTTPError(resp.StatusCode, string(body))
		}

		contentType := resp.Header.Get("Content-Type")
		return &AudioProxyResponse{
			ContentType: contentType,
			Body:        body,
		}, nil
	}

	rateCtx := scheduler.WithRateLimitUsage(ctx, buildAudioRateLimitUsage(input))
	result, err := s.cascadeController.ExecuteWithStrategy(rateCtx, operation, input.Model, scheduler.StrategyCostFirst, executor)
	if err != nil {
		return nil, err
	}
	if result == nil || result.Provider == nil || result.Response == nil {
		return nil, errors.New("调度结果为空")
	}

	proxyResp, ok := result.Response.(*AudioProxyResponse)
	if !ok || proxyResp == nil {
		return nil, errors.New("音频响应解析失败")
	}

	latency := int(time.Since(startTime).Milliseconds())
	provider := result.Provider

	downstreamCost := estimatedCost
	upstreamCost := decimal.Zero
	if provider != nil {
		if charged, cost, ok := s.resolvePricingRuleCosts(ctx, provider.ID, operation); ok {
			downstreamCost = charged
			upstreamCost = cost
		}
	}

	if downstreamCost.GreaterThan(decimal.Zero) {
		if token.User != nil {
			if err := DeductUserBalance(s.userRepo, token.UserID, downstreamCost, &token.User.Balance); err != nil {
				s.createAudioLog(token, provider, input.Model, operation, false, decimal.Zero, decimal.Zero, latency, model.LogStatusError, err.Error())
				return nil, err
			}
		}
		if token.RemainQuota != nil {
			if err := s.tokenService.DeductQuota(token.ID, downstreamCost); err != nil {
				s.createAudioLog(token, provider, input.Model, operation, false, decimal.Zero, decimal.Zero, latency, model.LogStatusError, err.Error())
				return nil, err
			}
		}
	}

	if s.healthTracker != nil {
		_ = s.healthTracker.RecordRequest(ctx, provider.ID, true, int64(latency), 0, 0, upstreamCost)
	}
	if provider.SelectedInstance != nil && s.instanceRepo != nil {
		_ = s.instanceRepo.IncrementStats(ctx, provider.SelectedInstance.ID, true, int64(latency))
	}

	s.createAudioLog(token, provider, input.Model, operation, false, downstreamCost, upstreamCost, latency, model.LogStatusSuccess, "")

	return proxyResp, nil
}

func (s *audioService) Speech(ctx context.Context, req *model.AudioSpeechRequest, token *model.Token, writer http.ResponseWriter) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil {
		return errors.New("请求不能为空")
	}
	if token == nil {
		return errors.New("token 不能为空")
	}
	if writer == nil {
		return errors.New("writer 不能为空")
	}

	startTime := time.Now()

	estimatedCost := decimal.Zero
	if pricing, err := s.modelRepo.GetPricing(req.Model); err == nil && pricing != nil {
		estimatedCost = pricing.InputPrice
	}

	if estimatedCost.GreaterThan(decimal.Zero) {
		if token.User != nil && token.User.Balance.LessThan(estimatedCost) {
			return &InsufficientFundsError{Needed: estimatedCost, Balance: token.User.Balance}
		}
		if token.RemainQuota != nil && !token.UnlimitedQuota && token.RemainQuota.LessThan(estimatedCost) {
			return &QuotaExceededError{Needed: estimatedCost, Remaining: *token.RemainQuota}
		}
	}

	executor := func(execCtx context.Context, provider *model.ModelProvider, onFirstChunk func()) error {
		if execCtx == nil {
			execCtx = context.Background()
		}
		if provider == nil || provider.Channel == nil {
			return fmt.Errorf("源头缺少 channel 信息")
		}

		upstreamModel := provider.UpstreamModelName
		if upstreamModel == "" {
			upstreamModel = req.Model
		}

		reqCopy := *req
		reqCopy.Model = upstreamModel
		reqBody, err := json.Marshal(reqCopy)
		if err != nil {
			return err
		}

		upstreamURL, err := buildOpenAIURL(provider.Channel.BaseURL, "/audio/speech")
		if err != nil {
			return err
		}

		upstreamReq, err := http.NewRequestWithContext(execCtx, http.MethodPost, upstreamURL, bytes.NewReader(reqBody))
		if err != nil {
			return err
		}
		upstreamReq.Header.Set("Content-Type", "application/json")
		upstreamReq.Header.Set("Authorization", "Bearer "+provider.Channel.APIKey)

		resp, err := s.httpClient.Do(upstreamReq)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return NewUpstreamHTTPError(resp.StatusCode, string(body))
		}

		if contentType := resp.Header.Get("Content-Type"); contentType != "" {
			writer.Header().Set("Content-Type", contentType)
		}

		flusher, _ := writer.(http.Flusher)
		buf := make([]byte, 32*1024)
		first := true

		for {
			n, rErr := resp.Body.Read(buf)
			if n > 0 {
				if _, wErr := writer.Write(buf[:n]); wErr != nil {
					return wErr
				}
				if flusher != nil {
					flusher.Flush()
				}
				if first {
					first = false
					if onFirstChunk != nil {
						onFirstChunk()
					}
				}
			}

			if rErr == io.EOF {
				break
			}
			if rErr != nil {
				return rErr
			}
		}

		return nil
	}

	rateCtx := scheduler.WithRateLimitUsage(ctx, buildAudioSpeechRateLimitUsage(req))
	result, err := s.cascadeController.ExecuteStreamWithFailover(rateCtx, model.OperationAudioSpeech, req.Model, scheduler.StrategyCostFirst, executor)
	if err != nil {
		return err
	}
	if result == nil || result.Provider == nil {
		return errors.New("调度结果缺少源头信息")
	}

	latency := int(time.Since(startTime).Milliseconds())
	provider := result.Provider

	downstreamCost := estimatedCost
	upstreamCost := decimal.Zero
	if provider != nil {
		if charged, cost, ok := s.resolvePricingRuleCosts(ctx, provider.ID, model.OperationAudioSpeech); ok {
			downstreamCost = charged
			upstreamCost = cost
		}
	}

	if downstreamCost.GreaterThan(decimal.Zero) {
		if token.User != nil {
			_ = s.userRepo.DeductBalance(token.UserID, downstreamCost)
		}
		if token.RemainQuota != nil {
			_ = s.tokenService.DeductQuota(token.ID, downstreamCost)
		}
	}

	if s.healthTracker != nil {
		_ = s.healthTracker.RecordRequest(ctx, provider.ID, true, int64(latency), 0, 0, upstreamCost)
	}
	if provider.SelectedInstance != nil && s.instanceRepo != nil {
		_ = s.instanceRepo.IncrementStats(ctx, provider.SelectedInstance.ID, true, int64(latency))
	}

	s.createAudioLog(token, provider, req.Model, model.OperationAudioSpeech, true, downstreamCost, upstreamCost, latency, model.LogStatusSuccess, "")
	return nil
}

func (s *audioService) createAudioLog(
	token *model.Token,
	provider *model.ModelProvider,
	modelID string,
	operation string,
	isStream bool,
	cost decimal.Decimal,
	upstreamCost decimal.Decimal,
	latency int,
	status model.LogStatus,
	errMsg string,
) {
	if s.logRepo == nil || token == nil {
		return
	}

	logItem := &model.Log{
		ID:           uuid.New(),
		UserID:       token.UserID,
		TokenID:      token.ID,
		Model:        modelID,
		TotalCost:    cost,
		UpstreamCost: upstreamCost,
		Duration:     latency,
		Status:       status,
		IsStream:     isStream,
		ErrorMessage: errMsg,
		Metadata: model.JSON{
			"operation": operation,
			"modality":  "audio",
		},
	}

	if provider != nil && provider.Channel != nil {
		logItem.ChannelID = provider.Channel.ID
	}

	_ = s.logRepo.Create(logItem)
}

func buildAudioRateLimitUsage(input *AudioMultipartInput) map[string]int64 {
	if input == nil {
		return nil
	}
	return map[string]int64{
		"request": 1,
	}
}

func buildAudioSpeechRateLimitUsage(req *model.AudioSpeechRequest) map[string]int64 {
	if req == nil {
		return nil
	}
	usage := map[string]int64{
		"request": 1,
	}
	if req.Input != "" {
		tokens := len(req.Input) / 4
		if tokens > 0 {
			usage["token"] = int64(tokens)
			usage["tokens"] = int64(tokens)
		}
	}
	return usage
}
