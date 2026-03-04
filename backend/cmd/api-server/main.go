package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nexus-api/internal/app"
	"nexus-api/internal/cache"
	"nexus-api/internal/config"
	"nexus-api/internal/database"

	"github.com/joho/godotenv"
)

// @title Nexus API Gateway
// @version 1.0
// @description API 聚合网关系统 - 提供 OpenAI 兼容的 API 接口
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 输入 Bearer {token}

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库连接
	if err := database.Init(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// 自动迁移数据库
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 初始化种子数据（创建默认管理员）
	if err := database.SeedAdminUser(cfg); err != nil {
		log.Printf("Warning: Failed to seed admin user: %v", err)
	}

	// 初始化 Redis 连接
	if err := cache.Init(cfg); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer cache.Close()

	// 获取数据库实例
	db := database.GetDB()

	// 创建路由（组合根）
	r, err := app.NewRouter(cfg, db)
	if err != nil {
		log.Fatalf("Failed to init router: %v", err)
	}

	/*
		// 初始化仓库层
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

		// ⭐ 新增：模型源头相关仓库
		modelProviderRepo := repository.NewModelProviderRepository(db)
		providerMetricsRepo := repository.NewProviderMetricsRepository(db)
		providerInstanceRepo := repository.NewProviderInstanceRepository(db)

		// 初始化对象存储
		objStore, err := storage.NewObjectStorage(cfg.OSS)
		if err != nil {
			log.Fatalf("Failed to init object storage: %v", err)
		}

		// 初始化服务层
		userService := service.NewUserService(userRepo, logRepo, cfg)
		tokenService := service.NewTokenService(tokenRepo, userRepo)
		channelService := service.NewChannelService(channelRepo)
		assetService := service.NewAssetService(cfg, assetRepo, objStore)
		generationService := service.NewGenerationService(generationRepo, userRepo, modelRepo, tokenService, cfg)
		orderService := service.NewOrderService(orderRepo, userRepo)
		logService := service.NewLogService(logRepo)
		modelService := service.NewModelService(modelRepo)
		alertService := service.NewAlertService(alertRepo)

		// ⭐ 新增：调度组件
		providerSelector := scheduler.NewProviderSelector(modelProviderRepo)
		circuitBreaker := scheduler.NewCircuitBreaker(modelProviderRepo, providerMetricsRepo, nil)
		healthTracker := scheduler.NewHealthTracker(modelProviderRepo, providerMetricsRepo, nil)
		costCalculator := scheduler.NewCostCalculator(modelRepo, modelProviderRepo, nil)

		// ⭐ 新增：缺失的调度组件
		instanceScheduler := scheduler.NewInstanceScheduler(providerInstanceRepo, providerMetricsRepo, nil)
		cascadeController := scheduler.NewCascadeController(providerSelector, instanceScheduler, circuitBreaker, nil)
		streamGuard := scheduler.NewStreamGuard(nil)
		runtimeStateStore := scheduler.NewRuntimeStateStore(nil)
		errorClassifier := scheduler.NewErrorClassifier()

		// ⭐ 新增：GatewayService（使用新的调度架构）
		gatewayService := service.NewGatewayService(
			channelService,
			tokenService,
			userRepo,
			logRepo,
			modelRepo,
			resourceAccountRepo,
			channelRepo,
			// ⭐ 调度组件
			providerSelector,
			instanceScheduler,
			cascadeController,
			circuitBreaker,
			streamGuard,
			healthTracker,
			costCalculator,
			runtimeStateStore,
			errorClassifier,
			modelProviderRepo,
			providerInstanceRepo,
		)

		// ⭐ 新增：模型源头服务
		modelProviderService := service.NewModelProviderService(
			modelProviderRepo,
			providerMetricsRepo,
			channelRepo,
			modelRepo,
			providerSelector,
			circuitBreaker,
			healthTracker,
		)

		// 初始化处理器
		userHandler := handler.NewUserHandler(userService, tokenService, orderService, logService, modelService, announcementService)
		gatewayHandler := handler.NewGatewayHandler(gatewayService)
		adminHandler := handler.NewAdminHandler(userService, channelService, orderService, logService, modelService, resourceAccountRepo, announcementRepo)
		alertHandler := handler.NewAlertHandler(alertService)
		assetHandler := handler.NewAssetHandler(assetService)
		objectHandler := handler.NewObjectHandler(cfg.OSS.LocalDir, cfg.OSS.SignSecret)
		generationHandler := handler.NewGenerationHandler(generationService)

		// ⭐ 新增：模型源头处理器
		modelProviderHandler := handler.NewModelProviderHandler(modelProviderService, costCalculator)
		providerInstanceHandler := handler.NewProviderInstanceHandler(providerInstanceRepo)

		// 设置 Gin 模式
		if cfg.Server.Mode == "release" {
			gin.SetMode(gin.ReleaseMode)
		}

		// 创建 Gin 引擎
		r := gin.New()

		// 全局中间件
		r.Use(gin.Recovery())
		r.Use(middleware.LoggerMiddleware())
		r.Use(middleware.CORSMiddleware())

		// 健康检查
		r.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Unix()})
		})

		// Swagger 文档
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

		// 本地对象存储访问（签名 URL）
		if cfg.OSS.Driver == "local" {
			r.GET("/objects/*key", objectHandler.GetObject)
		}

		// 公开资源访问（网站素材）
		r.GET("/assets/:id", assetHandler.RedirectSiteAsset)

		// Gateway API 路由 (OpenAI 兼容)
		v1 := r.Group("/v1")
		{
			v1.Use(middleware.GatewayAuthMiddleware(tokenService))
			v1.Use(middleware.GatewayRateLimitMiddleware(cfg.RateLimit))

			// 聊天和文本完成
			v1.POST("/chat/completions", gatewayHandler.ChatCompletions)
			v1.POST("/completions", gatewayHandler.Completions)
			v1.POST("/embeddings", gatewayHandler.Embeddings)
			v1.GET("/models", gatewayHandler.ListModels)
			v1.GET("/models/:model", gatewayHandler.GetModel)

			// 图片生成
			v1.POST("/images/generations", generationHandler.ImageGenerations)

			// 视频生成
			v1.POST("/videos/generations", generationHandler.VideoGenerations)

			// 生成任务管理
			v1.GET("/generations", generationHandler.ListUserTasks)
			v1.GET("/generations/:id", generationHandler.GetTaskStatus)
		}

		// 用户 API 路由
		api := r.Group("/api")
		{
			// 公开路由 - 用户认证（与前端路径一致）
			api.POST("/user/register", userHandler.Register)
			api.POST("/user/login", userHandler.Login)

			// 公开 API 路由（不需要认证）
			publicAPI := api.Group("/public")
			{
				// 获取可用模型列表
				publicAPI.GET("/models", userHandler.GetPublicModels)

				// 获取公告列表
				publicAPI.GET("/announcements", userHandler.GetPublicAnnouncements)
			}

			// 需要认证的用户路由
			userRoutes := api.Group("/user")
			userRoutes.Use(middleware.JWTAuthMiddleware(cfg.JWT.Secret))
			{
				// 用户资料
				userRoutes.GET("/profile", userHandler.GetProfile)
				userRoutes.PUT("/profile", userHandler.UpdateProfile)
				userRoutes.PUT("/password", userHandler.ChangePassword)
				userRoutes.PUT("/notifications", userHandler.UpdateNotifications)

				// 仪表盘
				userRoutes.GET("/dashboard", userHandler.GetDashboard)

				// API Token 管理
				userRoutes.GET("/tokens", userHandler.ListTokens)
				userRoutes.POST("/tokens", userHandler.CreateToken)
				userRoutes.GET("/tokens/:id", userHandler.GetToken)
				userRoutes.PUT("/tokens/:id", userHandler.UpdateToken)
				userRoutes.DELETE("/tokens/:id", userHandler.DeleteToken)
				userRoutes.PUT("/tokens/:id/status", userHandler.UpdateTokenStatus)

				// 订单管理
				userRoutes.GET("/orders", userHandler.ListOrders)
				userRoutes.POST("/orders", userHandler.CreateOrder)
				userRoutes.GET("/orders/:id", userHandler.GetOrder)
				userRoutes.POST("/orders/:id/cancel", userHandler.CancelOrder)

				// 日志管理
				userRoutes.GET("/logs", userHandler.ListLogs)
				userRoutes.GET("/logs/stats", userHandler.GetLogStats)

				// 账单管理
				userRoutes.GET("/billing/overview", userHandler.GetBillingOverview)
				userRoutes.GET("/billing/consumption", userHandler.GetConsumptionDetails)

				// 用户文件上传（测试上传）
				userRoutes.POST("/assets/uploads", assetHandler.UploadUserUpload)
				userRoutes.GET("/assets/:id/url", assetHandler.GetUserAssetURL)
				userRoutes.GET("/assets/:id", assetHandler.RedirectUserAsset)
			}
		}

		// 管理员公开路由（不需要认证）
		api.POST("/admin/login", adminHandler.AdminLogin)

		// 管理员 API 路由（需要认证）
		adminAPI := r.Group("/api/admin")
		adminAPI.Use(middleware.JWTAuthMiddleware(cfg.JWT.Secret))
		adminAPI.Use(middleware.AdminOnlyMiddleware())
		{
			// 仪表板
			adminAPI.GET("/dashboard", adminHandler.GetDashboard)
			adminAPI.GET("/dashboard/system", adminHandler.GetSystemMonitor)
			adminAPI.GET("/dashboard/alerts", adminHandler.GetAlerts)

			// 渠道管理
			adminAPI.GET("/channels", adminHandler.ListChannels)
			adminAPI.POST("/channels", adminHandler.CreateChannel)
			adminAPI.GET("/channels/:id", adminHandler.GetChannel)
			adminAPI.PUT("/channels/:id", adminHandler.UpdateChannel)
			adminAPI.DELETE("/channels/:id", adminHandler.DeleteChannel)
			adminAPI.POST("/channels/:id/test", adminHandler.TestChannel)

			// 模型管理
			adminAPI.GET("/models", adminHandler.ListModels)
			adminAPI.POST("/models", adminHandler.CreateModel)
			adminAPI.GET("/models/:id", adminHandler.GetModel)
			adminAPI.PUT("/models/:id", adminHandler.UpdateModel)
			adminAPI.DELETE("/models/:id", adminHandler.DeleteModel)
			adminAPI.PUT("/models/:id/status", adminHandler.UpdateModelStatus)

			// 用户管理
			adminAPI.GET("/users", adminHandler.ListUsers)
			adminAPI.POST("/users", adminHandler.CreateUser)
			adminAPI.GET("/users/:id", adminHandler.GetUser)
			adminAPI.PUT("/users/:id", adminHandler.UpdateUser)
			adminAPI.DELETE("/users/:id", adminHandler.DeleteUser)
			adminAPI.PUT("/users/:id/status", adminHandler.UpdateUserStatus)
			adminAPI.PUT("/users/:id/balance", adminHandler.UpdateUserBalance)

			// 日志管理
			adminAPI.GET("/logs", adminHandler.ListLogs)
			adminAPI.GET("/logs/stats", adminHandler.GetLogStats)

			// 订单管理
			adminAPI.GET("/orders", adminHandler.ListOrders)
			adminAPI.GET("/orders/:id", adminHandler.GetOrder)
			adminAPI.PUT("/orders/:id/status", adminHandler.UpdateOrderStatus)

			// 公告管理
			adminAPI.GET("/announcements", adminHandler.ListAnnouncements)
			adminAPI.POST("/announcements", adminHandler.CreateAnnouncement)
			adminAPI.GET("/announcements/:id", adminHandler.GetAnnouncement)
			adminAPI.PUT("/announcements/:id", adminHandler.UpdateAnnouncement)
			adminAPI.DELETE("/announcements/:id", adminHandler.DeleteAnnouncement)
			adminAPI.POST("/announcements/:id/publish", adminHandler.PublishAnnouncement)
			adminAPI.POST("/announcements/:id/archive", adminHandler.ArchiveAnnouncement)

			// 资源账户管理
			adminAPI.GET("/resource-accounts", adminHandler.ListResourceAccounts)
			adminAPI.POST("/resource-accounts", adminHandler.CreateResourceAccount)
			adminAPI.GET("/resource-accounts/:id", adminHandler.GetResourceAccount)
			adminAPI.PUT("/resource-accounts/:id", adminHandler.UpdateResourceAccount)
			adminAPI.DELETE("/resource-accounts/:id", adminHandler.DeleteResourceAccount)
			adminAPI.POST("/resource-accounts/:id/refresh", adminHandler.RefreshResourceAccount)
			adminAPI.GET("/resource-accounts/stats", adminHandler.GetResourceAccountStats)

			// 系统统计
			adminAPI.GET("/stats", adminHandler.GetSystemStats)

			// 系统设置
			adminAPI.PUT("/settings/:section", adminHandler.SaveSettings)

			// 告警管理
			adminAPI.GET("/alerts", alertHandler.ListAlerts)
			adminAPI.GET("/alerts/:id", alertHandler.GetAlert)
			adminAPI.PUT("/alerts/:id/resolve", alertHandler.ResolveAlert)
			adminAPI.GET("/alerts/stats", alertHandler.GetAlertStats)
			adminAPI.GET("/alerts/active", alertHandler.GetActiveAlerts)

			// 网站素材上传
			adminAPI.POST("/assets/site", assetHandler.UploadSiteMaterial)

			// ⭐ 新增：模型源头管理
			modelProviderHandler.RegisterRoutes(adminAPI)

			// ⭐ 新增：源头实例管理
			providerInstanceHandler.RegisterRoutes(adminAPI)

			// ⭐ 新增：调度监控
			handler.RegisterDispatchRoutes(adminAPI, db)
		}

	*/

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 启动服务器
	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		log.Printf("Swagger docs available at http://localhost:%s/swagger/index.html", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
