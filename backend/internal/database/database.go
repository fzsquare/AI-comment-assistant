package database

import (
	"log"

	"ppk/backend/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Connect(cfg config.Config) *gorm.DB {
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	return db
}
