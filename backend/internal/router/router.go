package router

import (
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

	authService := &service.AuthService{DB: db, Config: cfg}

	// 主生成器：有 agent 服务地址就走 Python 文案 agent，否则回退内置 Mock。
	var generator service.ReviewGenerator = &service.MockReviewGenerator{}
	if cfg.AgentServiceURL != "" {
		generator = service.NewAgentReviewGenerator(cfg.AgentServiceURL, cfg.AgentMinGrade, cfg.AgentInternalToken)
	}
	reviewPoolService := &service.ReviewPoolService{
		DB:        db,
		Generator: generator,
		Fallback:  &service.MockReviewGenerator{}, // 池空且 agent 不可用时即时兜底，避免白屏
	}

	merchant := merchantHandler.NewHandler(db, cfg, authService, reviewPoolService)
	admin := adminHandler.NewHandler(db, cfg, authService, reviewPoolService)
	public := publicHandler.NewHandler(db, reviewPoolService)

	api := r.Group("/api")
	{
		merchant.Register(api)
		admin.Register(api)
		public.Register(api)
	}

	return r
}
