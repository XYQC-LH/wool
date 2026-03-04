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

	"nexus-api/internal/config"
	"nexus-api/internal/database"
	"nexus-api/internal/repository"
	"nexus-api/internal/service"

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

	db := database.GetDB()
	metricsRepo := repository.NewProviderMetricsRepository(db)

	workerCfg := service.DefaultProviderMetricsWorkerConfig()
	workerCfg.PollInterval = time.Duration(getEnvInt("METRICS_WORKER_POLL_SECONDS", int(workerCfg.PollInterval.Seconds()))) * time.Second
	workerCfg.Lookback = time.Duration(getEnvInt("METRICS_WORKER_LOOKBACK_SECONDS", int(workerCfg.Lookback.Seconds()))) * time.Second

	worker, err := service.NewProviderMetricsWorker(db, metricsRepo, workerCfg)
	if err != nil {
		log.Fatalf("Failed to init metrics worker: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := worker.Run(ctx); err != nil {
			log.Printf("Metrics worker exited with error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down metrics worker...")
	cancel()

	time.Sleep(2 * time.Second)
	log.Println("Metrics worker exited")
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
