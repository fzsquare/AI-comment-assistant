package router

import (
	"context"
	"os"
	"strings"
	"time"

	"ppk/backend/internal/config"
	adminHandler "ppk/backend/internal/handler/admin"
	merchantHandler "ppk/backend/internal/handler/merchant"
	publicHandler "ppk/backend/internal/handler/public"
	"ppk/backend/internal/middleware"
	"ppk/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(cfg config.Config, db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// 商家上传图片的本地目录 + 静态访问
	_ = os.MkdirAll(cfg.UploadDir, 0o755)
	r.Static("/uploads", cfg.UploadDir)

	authService := &service.AuthService{DB: db, Config: cfg}
	reviewPoolService := buildReviewPoolService(cfg, db)
	reviewCrawlService := service.NewReviewCrawlService(db, cfg)
	reviewCrawlService.StartScheduler(context.Background())

	merchant := merchantHandler.NewHandler(db, cfg, authService, reviewPoolService, reviewCrawlService)
	admin := adminHandler.NewHandler(db, cfg, authService, reviewPoolService, reviewCrawlService)
	public := publicHandler.NewHandler(db, reviewPoolService)

	api := r.Group("/api")
	{
		merchant.Register(api)
		admin.Register(api)
		public.Register(api)
	}

	return r
}

func buildReviewPoolService(cfg config.Config, db *gorm.DB) *service.ReviewPoolService {
	var generator service.ReviewGenerator = service.NewUnavailableReviewGenerator("AGENT_SERVICE_URL is required")
	if strings.TrimSpace(cfg.AgentServiceURL) != "" {
		generator = service.NewAgentReviewGeneratorWithOptions(
			cfg.AgentServiceURL,
			cfg.AgentMinGrade,
			cfg.AgentInternalToken,
			time.Duration(cfg.AgentHTTPTimeoutSeconds)*time.Second,
			cfg.AgentGenerationBatchSize,
		)
	}
	return &service.ReviewPoolService{
		DB:            db,
		Generator:     generator,
		SessionSecret: cfg.JWTSecret,
	}
}
