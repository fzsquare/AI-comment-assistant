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
	r.Use(middleware.CORS())

	authService := &service.AuthService{DB: db, Config: cfg}
	reviewPoolService := &service.ReviewPoolService{DB: db, Generator: &service.MockReviewGenerator{}}

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
