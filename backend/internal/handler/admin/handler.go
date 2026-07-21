package admin

import (
	"ppk/backend/internal/config"
	"ppk/backend/internal/service"

	"gorm.io/gorm"
)

type Handler struct {
	DB          *gorm.DB
	Config      config.Config
	Auth        *service.AuthService
	ReviewPool  *service.ReviewPoolService
	ReviewCrawl *service.ReviewCrawlService
}

func NewHandler(db *gorm.DB, cfg config.Config, auth *service.AuthService, reviewPool *service.ReviewPoolService, reviewCrawl *service.ReviewCrawlService) *Handler {
	return &Handler{DB: db, Config: cfg, Auth: auth, ReviewPool: reviewPool, ReviewCrawl: reviewCrawl}
}
