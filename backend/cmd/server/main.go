package main

import (
	"log"

	"ppk/backend/internal/config"
	"ppk/backend/internal/database"
	"ppk/backend/internal/router"
)

func main() {
	cfg := config.Load()
	db := database.Connect(cfg)
	r := router.SetupRouter(cfg, db)

	log.Printf("server listening on :%s", cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}
