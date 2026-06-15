package config

import "os"

type Config struct {
	AppPort   string
	MySQLDSN  string
	JWTSecret string
}

func Load() Config {
	return Config{
		AppPort:   getEnv("APP_PORT", "8080"),
		MySQLDSN:  getEnv("MYSQL_DSN", "root:111111@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local"),
		JWTSecret: getEnv("JWT_SECRET", "ppk-dev-secret"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
