package config

import "os"

type Config struct {
	AppPort   string
	MySQLDSN  string
	JWTSecret string
	// 文案 agent 服务（Python）地址；为空则回退到内置 Mock 生成器
	AgentServiceURL string
	// 入池最低质量等级（S/A/B/C/D，对应约束手册 6.1），默认 B
	AgentMinGrade string
}

func Load() Config {
	return Config{
		AppPort:         getEnv("APP_PORT", "8080"),
		MySQLDSN:        getEnv("MYSQL_DSN", "root:111111@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local"),
		JWTSecret:       getEnv("JWT_SECRET", "ppk-dev-secret"),
		AgentServiceURL: getEnv("AGENT_SERVICE_URL", "http://127.0.0.1:8090"),
		AgentMinGrade:   getEnv("AGENT_MIN_GRADE", "B"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
