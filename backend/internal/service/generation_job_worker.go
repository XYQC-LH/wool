package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type GenerationJobWorkerConfig struct {
	PollInterval time.Duration
	BatchSize    int
	Concurrency  int
	StaleAfter   time.Duration
}

func DefaultGenerationJobWorkerConfig() GenerationJobWorkerConfig {
	return GenerationJobWorkerConfig{
		PollInterval: 2 * time.Second,
		BatchSize:    5,
		Concurrency:  2,
		StaleAfter:   15 * time.Minute,
	}
}

// GenerationJobWorker 閻劋绨径鍕倞 generation_tasks閿涘牆娴橀悧?鐟欏棝顣堕敍澶婄磽濮濄儰缍旀稉姘モ偓?//
// GenerationJobWorker 轮询 generation_tasks 并驱动异步任务执行。
type GenerationJobWorker struct {
	svc       *generationService
	tokenRepo repository.TokenRepository
	cfg       GenerationJobWorkerConfig
}

func NewGenerationJobWorker(genSvc GenerationService, tokenRepo repository.TokenRepository, cfg GenerationJobWorkerConfig) (*GenerationJobWorker, error) {
	if genSvc == nil {
		return nil, fmt.Errorf("generation service 娑撳秷鍏樻稉铏光敄")
	}
	if tokenRepo == nil {
		return nil, fmt.Errorf("tokenRepo 娑撳秷鍏樻稉铏光敄")
	}

	svc, ok := genSvc.(*generationService)
	if !ok {
		return nil, fmt.Errorf("generation service 鐎圭偟骞囨稉宥嗘暜閹镐礁绱撳?worker")
	}

	if cfg.PollInterval <= 0 {
		cfg.PollInterval = DefaultGenerationJobWorkerConfig().PollInterval
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultGenerationJobWorkerConfig().BatchSize
	}
	if cfg.BatchSize > 100 {
		cfg.BatchSize = 100
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = DefaultGenerationJobWorkerConfig().Concurrency
	}
	if cfg.Concurrency > 20 {
		cfg.Concurrency = 20
	}

	return &GenerationJobWorker{
		svc:       svc,
		tokenRepo: tokenRepo,
		cfg:       cfg,
	}, nil
}

func (w *GenerationJobWorker) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	log.Printf("[job-worker] started poll=%s batch=%d concurrency=%d staleAfter=%s", w.cfg.PollInterval, w.cfg.BatchSize, w.cfg.Concurrency, w.cfg.StaleAfter)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[job-worker] stopping: %v", ctx.Err())
			return nil
		case <-ticker.C:
			if err := w.runOnce(ctx); err != nil {
				log.Printf("[job-worker] runOnce error: %v", err)
			}
		}
	}
}

func (w *GenerationJobWorker) runOnce(ctx context.Context) error {
	if w.cfg.StaleAfter > 0 {
		staleBefore := time.Now().Add(-w.cfg.StaleAfter)
		_, _ = w.svc.generationRepo.RequeueStaleProcessingTasks(ctx, staleBefore)
	}

	tasks, err := w.svc.generationRepo.ClaimPendingTasks(ctx, w.cfg.BatchSize)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		return nil
	}

	sem := make(chan struct{}, w.cfg.Concurrency)
	var wg sync.WaitGroup

	for i := range tasks {
		task := tasks[i]
		if task == nil {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(t *model.GenerationTask) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := w.processTask(ctx, t); err != nil {
				log.Printf("[job-worker] task %s failed: %v", t.ID.String(), err)
			}
		}(task)
	}

	wg.Wait()
	return nil
}

func (w *GenerationJobWorker) processTask(ctx context.Context, task *model.GenerationTask) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if task == nil {
		return nil
	}

	startTime := time.Now()

	if task.TokenID == uuid.Nil {
		err := errors.New("娴犺濮熺紓鍝勭毌 token_id")
		_ = w.svc.failTask(task, startTime, err, task.Progress)
		return err
	}

	token, err := w.tokenRepo.GetByID(task.TokenID)
	if err != nil {
		_ = w.svc.failTask(task, startTime, err, task.Progress)
		return err
	}
	if token == nil || !token.IsValid() {
		err := errors.New("token is invalid or expired")
		_ = w.svc.failTask(task, startTime, err, task.Progress)
		return err
	}

	switch task.Type {
	case model.GenerationTypeImage:
		return w.processImageTask(ctx, task, token, startTime)
	case model.GenerationTypeVideo:
		return w.processVideoTask(ctx, task, token, startTime)
	default:
		err := fmt.Errorf("娑撳秵鏁幐浣烘畱娴犺濮熺猾璇茬€? %s", task.Type)
		_ = w.svc.failTask(task, startTime, err, task.Progress)
		return err
	}
}

func (w *GenerationJobWorker) processImageTask(ctx context.Context, task *model.GenerationTask, token *model.Token, startTime time.Time) error {
	req := imageRequestFromTask(task)
	if req == nil {
		err := errors.New("娴犺濮熼崣鍌涙殶鐟欙絾鐎芥径杈Е")
		_ = w.svc.failTask(task, startTime, err, task.Progress)
		return err
	}

	modelPricing, err := w.svc.modelRepo.GetPricing(req.Model)
	if err != nil {
		modelPricing = &model.ModelPricing{
			ModelID:     req.Model,
			InputPrice:  decimal.NewFromFloat(0.01),
			OutputPrice: decimal.NewFromFloat(0.01),
			PriceUnit:   1,
		}
	}

	estimatedCost := w.svc.calculateImageCost(modelPricing, req)
	if err := w.svc.ensureSufficientBalance(ctx, token, estimatedCost); err != nil {
		_ = w.svc.failTask(task, startTime, err, task.Progress)
		return err
	}

	return w.svc.processImageTask(ctx, req, token, task, modelPricing, startTime, true)
}

func (w *GenerationJobWorker) processVideoTask(ctx context.Context, task *model.GenerationTask, token *model.Token, startTime time.Time) error {
	req := videoRequestFromTask(task)
	if req == nil {
		err := errors.New("娴犺濮熼崣鍌涙殶鐟欙絾鐎芥径杈Е")
		_ = w.svc.failTask(task, startTime, err, task.Progress)
		return err
	}

	modelPricing, err := w.svc.modelRepo.GetPricing(req.Model)
	if err != nil {
		modelPricing = &model.ModelPricing{
			ModelID:     req.Model,
			InputPrice:  decimal.NewFromFloat(0.1),
			OutputPrice: decimal.NewFromFloat(0.1),
			PriceUnit:   1,
		}
	}

	estimatedCost := w.svc.calculateVideoCost(modelPricing, req)
	if err := w.svc.ensureSufficientBalance(ctx, token, estimatedCost); err != nil {
		_ = w.svc.failTask(task, startTime, err, task.Progress)
		return err
	}

	return w.svc.processVideoTask(ctx, req, token, task, modelPricing, startTime, true)
}

func imageRequestFromTask(task *model.GenerationTask) *model.ImageGenerationRequest {
	if task == nil {
		return nil
	}

	req := &model.ImageGenerationRequest{
		Model:  task.Model,
		Prompt: task.Prompt,
	}

	if task.Params == nil {
		return req
	}

	req.Size = getString(task.Params, "size")
	req.AspectRatio = getString(task.Params, "aspect_ratio")
	req.Resolution = getString(task.Params, "resolution")
	req.ResponseFormat = getString(task.Params, "response_format")
	req.N = getInt(task.Params, "n")
	req.Seed = getInt(task.Params, "seed")
	req.Watermark = getBool(task.Params, "watermark")
	req.Image = getString(task.Params, "image")

	if urls := getStringSlice(task.Params, "urls"); len(urls) > 0 {
		req.URLs = urls
	}

	return req
}

func videoRequestFromTask(task *model.GenerationTask) *model.VideoGenerationRequest {
	if task == nil {
		return nil
	}

	req := &model.VideoGenerationRequest{
		Model:  task.Model,
		Prompt: task.Prompt,
	}

	if task.Params == nil {
		return req
	}

	req.AspectRatio = getString(task.Params, "aspect_ratio")
	req.Duration = getInt(task.Params, "duration")
	req.Size = getString(task.Params, "size")
	req.ImageURL = getString(task.Params, "image_url")

	return req
}

func getString(m model.JSON, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func getInt(m model.JSON, key string) int {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float32:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func getBool(m model.JSON, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func getStringSlice(m model.JSON, key string) []string {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}

	switch vv := v.(type) {
	case []string:
		return vv
	case []interface{}:
		out := make([]string, 0, len(vv))
		for _, item := range vv {
			if item == nil {
				continue
			}
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
				continue
			}
			out = append(out, fmt.Sprintf("%v", item))
		}
		return out
	default:
		return nil
	}
}
