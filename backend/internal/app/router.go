package app

import (
	"net/http"
	"time"

	"nexus-api/internal/config"
	"nexus-api/internal/handler"
	"nexus-api/internal/middleware"
	"nexus-api/internal/repository"
	"nexus-api/internal/service"
	"nexus-api/internal/service/scheduler"
	"nexus-api/internal/storage"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// NewRouter 创建并装配 Gin 路由（组合根）
func NewRouter(cfg *config.Config, db *gorm.DB) (*gin.Engine, error) {
	// ==================== Repository ====================
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	channelRepo := repository.NewChannelRepository(db)
	modelRepo := repository.NewModelRepository(db)
	logRepo := repository.NewLogRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	resourceAccountRepo := repository.NewResourceAccountRepository(db)
	announcementRepo := repository.NewAnnouncementRepository(db)
	assetRepo := repository.NewAssetRepository(db)
	generationRepo := repository.NewGenerationRepository(db)
	alertRepo := repository.NewAlertRepository(db)
	systemSettingRepo := repository.NewSystemSettingRepository(db)
	modelAliasRepo := repository.NewModelAliasRepository(db)
	idempotencyKeyRepo := repository.NewIdempotencyKeyRepository(db)

	modelProviderRepo := repository.NewModelProviderRepository(db)
	providerMetricsRepo := repository.NewProviderMetricsRepository(db)
	providerInstanceRepo := repository.NewProviderInstanceRepository(db)
	modelCapabilityRepo := repository.NewModelCapabilityRepository(db)
	modelRouteRepo := repository.NewModelRouteRepository(db)
	providerCapabilityRepo := repository.NewProviderCapabilityRepository(db)
	providerPricingRuleRepo := repository.NewProviderPricingRuleRepository(db)
	providerRateLimitRuleRepo := repository.NewProviderRateLimitRuleRepository(db)
	dispatchAttemptRepo := repository.NewDispatchAttemptRepository(db)

	// ==================== Infra ====================
	objStore, err := storage.NewObjectStorage(cfg.OSS)
	if err != nil {
		return nil, err
	}

	// ==================== Scheduler (Model Provider Layer) ====================
	runtimeStateStore := scheduler.NewRuntimeStateStore()
	providerSelector := scheduler.NewProviderSelector(modelProviderRepo, providerPricingRuleRepo)
	modelAggregator := scheduler.NewModelAggregator(modelProviderRepo, modelAliasRepo)
	routeResolver := scheduler.NewRouteResolver(modelRouteRepo)
	capabilityMatcher := scheduler.NewCapabilityMatcher(providerCapabilityRepo)
	sourceAdapterRegistry := scheduler.NewSourceAdapterRegistry(
		scheduler.NewOpenAICompatibleAdapter(),
	)
	providerRateLimiter := scheduler.NewProviderRateLimiter(providerRateLimitRuleRepo, nil)
	commitGuard := scheduler.NewCommitGuard(runtimeStateStore, nil)
	streamGuard := commitGuard
	errorClassifier := scheduler.NewErrorClassifier(nil)
	instanceScheduler := scheduler.NewInstanceScheduler(providerInstanceRepo, providerRateLimitRuleRepo, runtimeStateStore, nil)
	circuitBreaker := scheduler.NewCircuitBreaker(modelProviderRepo, providerMetricsRepo, runtimeStateStore, nil)
	costCalculator := scheduler.NewCostCalculator(modelRepo, modelProviderRepo, providerMetricsRepo, nil)
	healthTracker := scheduler.NewHealthTracker(modelProviderRepo, providerMetricsRepo, nil)
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
	userService := service.NewUserService(userRepo, logRepo, cfg)
	tokenService := service.NewTokenService(tokenRepo, userRepo, logRepo)
	channelService := service.NewChannelService(channelRepo)
	announcementService := service.NewAnnouncementService(announcementRepo)
	assetService := service.NewAssetService(cfg, assetRepo, objStore)
	generationService := service.NewGenerationService(generationRepo, idempotencyKeyRepo, providerPricingRuleRepo, logRepo, userRepo, modelRepo, modelProviderRepo, tokenService, cascadeController, commitGuard, objStore, cfg)
	audioService := service.NewAudioService(userRepo, tokenService, modelRepo, providerPricingRuleRepo, logRepo, providerInstanceRepo, healthTracker, cascadeController)
	orderService := service.NewOrderService(orderRepo, userRepo)
	logService := service.NewLogService(logRepo)
	modelService := service.NewModelService(modelRepo)
	alertService := service.NewAlertService(alertRepo)
	settingsService := service.NewSettingsService(systemSettingRepo)

	// 主网关实现：Gateway v2
	gatewayService := service.NewGatewayServiceV2(
		channelService,
		tokenService,
		userRepo,
		logRepo,
		modelRepo,
		resourceAccountRepo,
		channelRepo,
		modelProviderRepo,
		providerInstanceRepo,
		providerSelector,
		instanceScheduler,
		cascadeController,
		circuitBreaker,
		streamGuard,
		healthTracker,
		costCalculator,
		errorClassifier,
	)

	modelProviderService := service.NewModelProviderService(
		modelProviderRepo,
		providerMetricsRepo,
		channelRepo,
		modelRepo,
		providerSelector,
		circuitBreaker,
		healthTracker,
	)

	// ==================== Handler ====================
	userHandler := handler.NewUserHandler(userService, tokenService, orderService, logService, modelService, announcementService)
	gatewayHandler := handler.NewGatewayHandler(gatewayService)
	adminHandler := handler.NewAdminHandler(userService, channelService, orderService, logService, modelService, resourceAccountRepo, announcementRepo, settingsService, alertService)
	alertHandler := handler.NewAlertHandler(alertService)
	assetHandler := handler.NewAssetHandler(assetService)
	objectHandler := handler.NewObjectHandler(cfg.OSS.LocalDir, cfg.OSS.SignSecret)
	generationHandler := handler.NewGenerationHandler(generationService)
	audioHandler := handler.NewAudioHandler(audioService)

	modelProviderHandler := handler.NewModelProviderHandler(modelProviderService, costCalculator)
	providerInstanceHandler := handler.NewProviderInstanceHandler(providerInstanceRepo, runtimeStateStore)
	modelCapabilityHandler := handler.NewModelCapabilityHandler(modelCapabilityRepo)
	providerPricingRuleHandler := handler.NewProviderPricingRuleHandler(providerPricingRuleRepo)
	providerRateLimitRuleHandler := handler.NewProviderRateLimitRuleHandler(providerRateLimitRuleRepo)

	// ==================== Router ====================
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.PrometheusMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Unix()})
	})
	r.GET("/metrics", middleware.MetricsHandler())

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	if cfg.OSS.Driver == "local" {
		r.GET("/objects/*key", objectHandler.GetObject)
	}

	r.GET("/assets/:id", assetHandler.RedirectSiteAsset)

	// Gateway API 路由 (OpenAI 兼容)
	v1 := r.Group("/v1")
	{
		v1.Use(middleware.GatewayAuthMiddleware(tokenService))
		v1.Use(middleware.GatewayRateLimitMiddleware(cfg.RateLimit))

		v1.POST("/chat/completions", gatewayHandler.ChatCompletions)
		v1.POST("/completions", gatewayHandler.Completions)
		v1.POST("/embeddings", gatewayHandler.Embeddings)
		v1.GET("/models", gatewayHandler.ListModels)
		v1.GET("/models/:model", gatewayHandler.GetModel)

		v1.POST("/images/generations", generationHandler.ImageGenerations)
		v1.POST("/videos/generations", generationHandler.VideoGenerations)
		v1.POST("/audio/transcriptions", audioHandler.Transcriptions)
		v1.POST("/audio/translations", audioHandler.Translations)
		v1.POST("/audio/speech", audioHandler.Speech)
		v1.GET("/generations", generationHandler.ListUserTasks)
		v1.GET("/generations/:id", generationHandler.GetTaskStatus)
	}

	// 用户 API 路由
	api := r.Group("/api")
	{
		api.POST("/user/register", userHandler.Register)
		api.POST("/user/login", userHandler.Login)

		publicAPI := api.Group("/public")
		{
			publicAPI.GET("/models", userHandler.GetPublicModels)
			publicAPI.GET("/announcements", userHandler.GetPublicAnnouncements)
		}

		userRoutes := api.Group("/user")
		userRoutes.Use(middleware.JWTAuthMiddleware(cfg.JWT.Secret))
		{
			userRoutes.GET("/profile", userHandler.GetProfile)
			userRoutes.PUT("/profile", userHandler.UpdateProfile)
			userRoutes.PUT("/password", userHandler.ChangePassword)
			userRoutes.PUT("/notifications", userHandler.UpdateNotifications)

			userRoutes.GET("/dashboard", userHandler.GetDashboard)

			userRoutes.GET("/tokens", userHandler.ListTokens)
			userRoutes.POST("/tokens", userHandler.CreateToken)
			userRoutes.GET("/tokens/:id", userHandler.GetToken)
			userRoutes.PUT("/tokens/:id", userHandler.UpdateToken)
			userRoutes.DELETE("/tokens/:id", userHandler.DeleteToken)
			userRoutes.PUT("/tokens/:id/status", userHandler.UpdateTokenStatus)

			userRoutes.GET("/orders", userHandler.ListOrders)
			userRoutes.POST("/orders", userHandler.CreateOrder)
			userRoutes.GET("/orders/:id", userHandler.GetOrder)
			userRoutes.GET("/orders/by-no/:order_no", userHandler.GetOrderByOrderNo)
			userRoutes.POST("/orders/:id/cancel", userHandler.CancelOrder)
			userRoutes.POST("/orders/by-no/:order_no/pay", userHandler.PayOrderByOrderNo)

			userRoutes.GET("/logs", userHandler.ListLogs)
			userRoutes.GET("/logs/stats", userHandler.GetLogStats)

			userRoutes.GET("/billing/overview", userHandler.GetBillingOverview)
			userRoutes.GET("/billing/consumption", userHandler.GetConsumptionDetails)

			userRoutes.POST("/assets/uploads", assetHandler.UploadUserUpload)
			userRoutes.GET("/assets/:id/url", assetHandler.GetUserAssetURL)
			userRoutes.GET("/assets/:id", assetHandler.RedirectUserAsset)
		}
	}

	api.POST("/admin/login", adminHandler.AdminLogin)

	adminAPI := r.Group("/api/admin")
	adminAPI.Use(middleware.JWTAuthMiddleware(cfg.JWT.Secret))
	adminAPI.Use(middleware.AdminOnlyMiddleware())
	{
		adminAPI.GET("/dashboard", adminHandler.GetDashboard)
		adminAPI.GET("/dashboard/system", adminHandler.GetSystemMonitor)
		adminAPI.GET("/dashboard/alerts", adminHandler.GetAlerts)

		adminAPI.GET("/channels", adminHandler.ListChannels)
		adminAPI.POST("/channels", adminHandler.CreateChannel)
		adminAPI.GET("/channels/:id", adminHandler.GetChannel)
		adminAPI.PUT("/channels/:id", adminHandler.UpdateChannel)
		adminAPI.DELETE("/channels/:id", adminHandler.DeleteChannel)
		adminAPI.POST("/channels/:id/test", adminHandler.TestChannel)
		adminAPI.PUT("/channels/:id/status", adminHandler.UpdateChannelStatus)

		adminAPI.GET("/models", adminHandler.ListModels)
		adminAPI.POST("/models", adminHandler.CreateModel)
		adminAPI.GET("/models/:id", adminHandler.GetModel)
		adminAPI.PUT("/models/:id", adminHandler.UpdateModel)
		adminAPI.DELETE("/models/:id", adminHandler.DeleteModel)
		adminAPI.PUT("/models/:id/status", adminHandler.UpdateModelStatus)

		adminAPI.GET("/users", adminHandler.ListUsers)
		adminAPI.POST("/users", adminHandler.CreateUser)
		adminAPI.GET("/users/:id", adminHandler.GetUser)
		adminAPI.GET("/users/:id/stats", adminHandler.GetUserStats)
		adminAPI.PUT("/users/:id", adminHandler.UpdateUser)
		adminAPI.DELETE("/users/:id", adminHandler.DeleteUser)
		adminAPI.PUT("/users/:id/status", adminHandler.UpdateUserStatus)
		adminAPI.PUT("/users/:id/balance", adminHandler.UpdateUserBalance)

		adminAPI.GET("/logs", adminHandler.ListLogs)
		adminAPI.GET("/logs/stats", adminHandler.GetLogStats)

		adminAPI.GET("/orders", adminHandler.ListOrders)
		handler.RegisterOrderStatsRoutes(adminAPI, db)
		adminAPI.GET("/orders/:id", adminHandler.GetOrder)
		adminAPI.PUT("/orders/:id/status", adminHandler.UpdateOrderStatus)

		adminAPI.GET("/announcements", adminHandler.ListAnnouncements)
		adminAPI.POST("/announcements", adminHandler.CreateAnnouncement)
		adminAPI.GET("/announcements/:id", adminHandler.GetAnnouncement)
		adminAPI.PUT("/announcements/:id", adminHandler.UpdateAnnouncement)
		adminAPI.DELETE("/announcements/:id", adminHandler.DeleteAnnouncement)
		adminAPI.POST("/announcements/:id/publish", adminHandler.PublishAnnouncement)
		adminAPI.POST("/announcements/:id/archive", adminHandler.ArchiveAnnouncement)

		adminAPI.GET("/resource-accounts", adminHandler.ListResourceAccounts)
		adminAPI.POST("/resource-accounts", adminHandler.CreateResourceAccount)
		adminAPI.GET("/resource-accounts/:id", adminHandler.GetResourceAccount)
		adminAPI.PUT("/resource-accounts/:id", adminHandler.UpdateResourceAccount)
		adminAPI.DELETE("/resource-accounts/:id", adminHandler.DeleteResourceAccount)
		adminAPI.POST("/resource-accounts/:id/refresh", adminHandler.RefreshResourceAccount)
		adminAPI.GET("/resource-accounts/stats", adminHandler.GetResourceAccountStats)

		adminAPI.GET("/stats", adminHandler.GetSystemStats)
		adminAPI.GET("/settings/:section", adminHandler.GetSettings)
		adminAPI.PUT("/settings/:section", adminHandler.SaveSettings)

		adminAPI.GET("/alerts", alertHandler.ListAlerts)
		adminAPI.GET("/alerts/:id", alertHandler.GetAlert)
		adminAPI.PUT("/alerts/:id/resolve", alertHandler.ResolveAlert)
		adminAPI.GET("/alerts/stats", alertHandler.GetAlertStats)
		adminAPI.GET("/alerts/active", alertHandler.GetActiveAlerts)

		adminAPI.POST("/assets/site", assetHandler.UploadSiteMaterial)

		modelProviderHandler.RegisterRoutes(adminAPI)
		providerInstanceHandler.RegisterRoutes(adminAPI)
		modelCapabilityHandler.RegisterRoutes(adminAPI)
		providerPricingRuleHandler.RegisterRoutes(adminAPI)
		providerRateLimitRuleHandler.RegisterRoutes(adminAPI)
		handler.RegisterTopologyRoutes(adminAPI, db)
		handler.RegisterDispatchRoutes(adminAPI, db)
		handler.RegisterFinanceRoutes(adminAPI, db)
	}

	return r, nil
}
