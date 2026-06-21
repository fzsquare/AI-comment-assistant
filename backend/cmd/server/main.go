package main

import (
	"log"

	"ppk/backend/internal/config"
	"ppk/backend/internal/database"
	"ppk/backend/internal/router"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}
	db := database.Connect(cfg)
	r := router.SetupRouter(cfg, db)

	addr := cfg.ListenAddress()
	log.Printf("server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
