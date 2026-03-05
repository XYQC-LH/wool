package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/config"
	"nexus-api/internal/database"
	"nexus-api/internal/repository"
	"nexus-api/internal/service"
	"nexus-api/internal/service/scheduler"
	"nexus-api/internal/storage"

	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库连接
	if err := database.Init(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// 自动迁移数据库（worker 也需要保证表存在）
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 初始化 Redis 连接
	if err := cache.Init(cfg); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer cache.Close()

	db := database.GetDB()

	// ==================== Repository ====================
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	logRepo := repository.NewLogRepository(db)
	modelRepo := repository.NewModelRepository(db)
	modelAliasRepo := repository.NewModelAliasRepository(db)
	modelCapabilityRepo := repository.NewModelCapabilityRepository(db)
	modelRouteRepo := repository.NewModelRouteRepository(db)
	providerCapabilityRepo := repository.NewProviderCapabilityRepository(db)
	modelProviderRepo := repository.NewModelProviderRepository(db)
	providerMetricsRepo := repository.NewProviderMetricsRepository(db)
	providerInstanceRepo := repository.NewProviderInstanceRepository(db)
	dispatchAttemptRepo := repository.NewDispatchAttemptRepository(db)
	generationRepo := repository.NewGenerationRepository(db)
	idempotencyRepo := repository.NewIdempotencyKeyRepository(db)
	pricingRuleRepo := repository.NewProviderPricingRuleRepository(db)
	rateLimitRuleRepo := repository.NewProviderRateLimitRuleRepository(db)

	// ==================== Infra ====================
	objStore, err := storage.NewObjectStorage(cfg.OSS)
	if err != nil {
		log.Fatalf("Failed to init object storage: %v", err)
	}

	// ==================== Scheduler ====================
	runtimeStateStore := scheduler.NewRuntimeStateStore()
	providerSelector := scheduler.NewProviderSelector(modelProviderRepo, pricingRuleRepo)
	modelAggregator := scheduler.NewModelAggregator(modelProviderRepo, modelAliasRepo)
	routeResolver := scheduler.NewRouteResolver(modelRouteRepo)
	capabilityMatcher := scheduler.NewCapabilityMatcher(providerCapabilityRepo)
	sourceAdapterRegistry := scheduler.NewSourceAdapterRegistry(
		scheduler.NewWebSocketTransportAdapter(),
		scheduler.NewGRPCTransportAdapter(),
		scheduler.NewAzureOpenAIAdapter(),
		scheduler.NewAnthropicAdapter(),
		scheduler.NewGoogleAdapter(),
		scheduler.NewOpenAICompatibleAdapter(),
	)
	providerRateLimiter := scheduler.NewProviderRateLimiter(rateLimitRuleRepo, nil)
	commitGuard := scheduler.NewCommitGuard(runtimeStateStore, nil)
	instanceScheduler := scheduler.NewInstanceScheduler(providerInstanceRepo, rateLimitRuleRepo, runtimeStateStore, nil)
	circuitBreaker := scheduler.NewCircuitBreaker(modelProviderRepo, providerMetricsRepo, runtimeStateStore, nil)
	cascadeController := scheduler.NewCascadeController(
		providerSelector,
		circuitBreaker,
		modelProviderRepo,
		providerMetricsRepo,
		instanceScheduler,
		providerInstanceRepo,
		commitGuard,
		modelAggregator,
		modelCapabilityRepo,
		capabilityMatcher,
		routeResolver,
		dispatchAttemptRepo,
		providerRateLimiter,
		nil,
		sourceAdapterRegistry,
	)

	// ==================== Service ====================
	tokenService := service.NewTokenService(tokenRepo, userRepo, logRepo)
	generationService := service.NewGenerationService(generationRepo, idempotencyRepo, pricingRuleRepo, logRepo, userRepo, modelRepo, modelProviderRepo, tokenService, cascadeController, commitGuard, objStore, cfg)

	workerCfg := service.DefaultGenerationJobWorkerConfig()
	workerCfg.PollInterval = time.Duration(getEnvInt("JOB_WORKER_POLL_SECONDS", int(workerCfg.PollInterval.Seconds()))) * time.Second
	workerCfg.BatchSize = getEnvInt("JOB_WORKER_BATCH_SIZE", workerCfg.BatchSize)
	workerCfg.Concurrency = getEnvInt("JOB_WORKER_CONCURRENCY", workerCfg.Concurrency)
	workerCfg.StaleAfter = time.Duration(getEnvInt("JOB_WORKER_STALE_SECONDS", int(workerCfg.StaleAfter.Seconds()))) * time.Second

	worker, err := service.NewGenerationJobWorker(generationService, tokenRepo, workerCfg)
	if err != nil {
		log.Fatalf("Failed to init job worker: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := worker.Run(ctx); err != nil {
			log.Printf("Job worker exited with error: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down job worker...")
	cancel()

	// 给 worker 一点时间收尾
	time.Sleep(2 * time.Second)
	log.Println("Job worker exited")
}

func getEnvInt(key string, defaultValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return v
}
