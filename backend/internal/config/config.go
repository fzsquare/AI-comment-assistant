package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	defaultAppEnv                         = "development"
	defaultAppHost                        = "127.0.0.1"
	defaultAppPort                        = "8080"
	defaultMySQLDSN                       = "ppk_dev:ppk_dev_password@tcp(127.0.0.1:3306)/ppk?charset=utf8mb4&parseTime=True&loc=Local"
	defaultJWTSecret                      = "dev-jwt-secret-change-me-32-bytes"
	defaultAgentServiceURL                = "http://127.0.0.1:8090"
	defaultAgentMinGrade                  = "B"
	defaultAgentInternalToken             = "dev-agent-internal-token-change-me"
	defaultAgentHTTPTimeoutSeconds        = 300
	defaultAgentGenerationBatchSize       = 2
	defaultAllowedOrigins                 = "http://localhost:3000,http://localhost:5173,http://127.0.0.1:3000,http://127.0.0.1:5173"
	defaultMaxReviewGenerateCount         = 50
	defaultReviewTargetCount              = 10
	defaultUploadDir                      = "./uploads"
	defaultReviewCrawlPollIntervalSeconds = 3
	defaultReviewCrawlPollMaxAttempts     = 40
	defaultReviewCrawlHTTPTimeoutSeconds  = 20
	defaultReviewCrawlMaxDownloadBytes    = 5 * 1024 * 1024
	minSecretLength                       = 32
)

type Config struct {
	AppEnv    string
	AppHost   string
	AppPort   string
	MySQLDSN  string
	JWTSecret string
	// 允许访问公开 API 的浏览器 Origin；生产环境必须显式配置且不能使用通配符。
	AllowedOrigins []string
	// 文案 agent 服务（Python）地址；为空时评论生成直接失败，不回退 mock
	AgentServiceURL string
	// Go 后端调用本地/内网 Python agent-service 时携带的内部令牌
	AgentInternalToken string
	// 入池最低质量等级（S/A/B/C/D，对应约束手册 6.1），默认 B
	AgentMinGrade string
	// Go 后端等待 agent-service 单批生成响应的 HTTP 超时秒数
	AgentHTTPTimeoutSeconds int
	// Go 后端调用 agent-service 时每批请求的评价数量
	AgentGenerationBatchSize int
	// 单次人工生成评价数量上限
	MaxReviewGenerateCount int
	// targetCount 缺省值
	DefaultReviewTargetCount int
	// 商家上传图片的本地保存目录
	UploadDir string
	// 上传图片/落地页对外访问的 URL 前缀；为空时用 PublicBasePath 或根路径相对地址
	PublicBaseURL string
	// 部署在子路径时的公开路径前缀，例如 /ppk；用于生成相对落地页和上传资源地址
	PublicBasePath string
	// 外部真实评论采集服务，仅后端访问；为空时采集功能不可用
	ReviewCrawlServiceURL          string
	ReviewCrawlPollIntervalSeconds int
	ReviewCrawlPollMaxAttempts     int
	ReviewCrawlHTTPTimeoutSeconds  int
	ReviewCrawlMaxDownloadBytes    int
}

func Load() Config {
	appEnv := getEnv("APP_ENV", defaultAppEnv)
	production := isProduction(appEnv)

	mysqlFallback := defaultMySQLDSN
	jwtFallback := defaultJWTSecret
	agentTokenFallback := defaultAgentInternalToken
	allowedOriginsFallback := defaultAllowedOrigins
	if production {
		mysqlFallback = ""
		jwtFallback = ""
		agentTokenFallback = ""
		allowedOriginsFallback = ""
	}

	return Config{
		AppEnv:                         appEnv,
		AppHost:                        getEnv("APP_HOST", defaultAppHost),
		AppPort:                        getEnv("APP_PORT", defaultAppPort),
		MySQLDSN:                       getEnv("MYSQL_DSN", mysqlFallback),
		JWTSecret:                      getEnv("JWT_SECRET", jwtFallback),
		AllowedOrigins:                 parseCSV(getEnv("ALLOWED_ORIGINS", allowedOriginsFallback)),
		AgentServiceURL:                strings.TrimRight(getEnv("AGENT_SERVICE_URL", defaultAgentServiceURL), "/"),
		AgentInternalToken:             getEnv("AGENT_INTERNAL_TOKEN", agentTokenFallback),
		AgentMinGrade:                  getEnv("AGENT_MIN_GRADE", defaultAgentMinGrade),
		AgentHTTPTimeoutSeconds:        getEnvInt("AGENT_HTTP_TIMEOUT_SECONDS", defaultAgentHTTPTimeoutSeconds),
		AgentGenerationBatchSize:       getEnvInt("AGENT_GENERATION_BATCH_SIZE", defaultAgentGenerationBatchSize),
		MaxReviewGenerateCount:         getEnvInt("MAX_REVIEW_GENERATE_COUNT", defaultMaxReviewGenerateCount),
		DefaultReviewTargetCount:       getEnvInt("DEFAULT_REVIEW_TARGET_COUNT", defaultReviewTargetCount),
		UploadDir:                      getEnv("UPLOAD_DIR", defaultUploadDir),
		PublicBaseURL:                  strings.TrimRight(getEnv("PUBLIC_BASE_URL", ""), "/"),
		PublicBasePath:                 normalizeBasePath(getEnv("PUBLIC_BASE_PATH", "")),
		ReviewCrawlServiceURL:          strings.TrimRight(getEnv("REVIEW_CRAWL_SERVICE_URL", ""), "/"),
		ReviewCrawlPollIntervalSeconds: getEnvInt("REVIEW_CRAWL_POLL_INTERVAL_SECONDS", defaultReviewCrawlPollIntervalSeconds),
		ReviewCrawlPollMaxAttempts:     getEnvInt("REVIEW_CRAWL_POLL_MAX_ATTEMPTS", defaultReviewCrawlPollMaxAttempts),
		ReviewCrawlHTTPTimeoutSeconds:  getEnvInt("REVIEW_CRAWL_HTTP_TIMEOUT_SECONDS", defaultReviewCrawlHTTPTimeoutSeconds),
		ReviewCrawlMaxDownloadBytes:    getEnvInt("REVIEW_CRAWL_MAX_DOWNLOAD_BYTES", defaultReviewCrawlMaxDownloadBytes),
	}
}

func (c Config) Validate() error {
	var problems []string
	production := isProduction(c.AppEnv)

	if strings.TrimSpace(c.AppPort) == "" {
		problems = append(problems, "APP_PORT is required")
	}
	if strings.TrimSpace(c.AppHost) == "" {
		problems = append(problems, "APP_HOST is required")
	}
	if strings.TrimSpace(c.MySQLDSN) == "" {
		problems = append(problems, "MYSQL_DSN is required")
	}
	if !strongSecret(c.JWTSecret) {
		problems = append(problems, fmt.Sprintf("JWT_SECRET must be at least %d characters", minSecretLength))
	}
	if c.AgentServiceURL != "" {
		if err := validateHTTPURL(c.AgentServiceURL); err != nil {
			problems = append(problems, "AGENT_SERVICE_URL must be an http(s) URL")
		}
	}
	if production || c.AgentServiceURL != "" {
		if !strongSecret(c.AgentInternalToken) {
			problems = append(problems, fmt.Sprintf("AGENT_INTERNAL_TOKEN must be at least %d characters", minSecretLength))
		}
	}
	if c.AgentHTTPTimeoutSeconds <= 0 {
		problems = append(problems, "AGENT_HTTP_TIMEOUT_SECONDS must be greater than 0")
	}
	if c.AgentGenerationBatchSize <= 0 {
		problems = append(problems, "AGENT_GENERATION_BATCH_SIZE must be greater than 0")
	}
	if c.MaxReviewGenerateCount <= 0 {
		problems = append(problems, "MAX_REVIEW_GENERATE_COUNT must be greater than 0")
	}
	if c.DefaultReviewTargetCount <= 0 || c.DefaultReviewTargetCount > c.MaxReviewGenerateCount {
		problems = append(problems, "DEFAULT_REVIEW_TARGET_COUNT must be between 1 and MAX_REVIEW_GENERATE_COUNT")
	}
	if strings.TrimSpace(c.UploadDir) == "" {
		problems = append(problems, "UPLOAD_DIR is required")
	}
	if c.PublicBaseURL != "" {
		if err := validateHTTPURL(c.PublicBaseURL); err != nil {
			problems = append(problems, "PUBLIC_BASE_URL must be an http(s) URL")
		}
	}
	if c.PublicBasePath != "" {
		if err := validateBasePath(c.PublicBasePath); err != nil {
			problems = append(problems, "PUBLIC_BASE_PATH must be a URL path prefix like /ppk")
		}
	}
	if c.ReviewCrawlServiceURL != "" {
		if err := validateHTTPURL(c.ReviewCrawlServiceURL); err != nil {
			problems = append(problems, "REVIEW_CRAWL_SERVICE_URL must be an http(s) URL")
		}
	}
	if c.ReviewCrawlPollIntervalSeconds <= 0 {
		problems = append(problems, "REVIEW_CRAWL_POLL_INTERVAL_SECONDS must be greater than 0")
	}
	if c.ReviewCrawlPollMaxAttempts <= 0 {
		problems = append(problems, "REVIEW_CRAWL_POLL_MAX_ATTEMPTS must be greater than 0")
	}
	if c.ReviewCrawlHTTPTimeoutSeconds <= 0 {
		problems = append(problems, "REVIEW_CRAWL_HTTP_TIMEOUT_SECONDS must be greater than 0")
	}
	if c.ReviewCrawlMaxDownloadBytes <= 0 {
		problems = append(problems, "REVIEW_CRAWL_MAX_DOWNLOAD_BYTES must be greater than 0")
	}

	for _, origin := range c.AllowedOrigins {
		if origin == "*" {
			problems = append(problems, "ALLOWED_ORIGINS cannot contain *")
			continue
		}
		if err := validateHTTPOrigin(origin); err != nil {
			problems = append(problems, fmt.Sprintf("ALLOWED_ORIGINS contains invalid origin %q", origin))
		}
	}
	if production && len(c.AllowedOrigins) == 0 {
		problems = append(problems, "ALLOWED_ORIGINS is required in production")
	}

	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func (c Config) ListenAddress() string {
	return net.JoinHostPort(c.AppHost, c.AppPort)
}

func (c Config) IsAllowedOrigin(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return false
	}
	for _, allowed := range c.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func parseCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func normalizeBasePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "/" {
		return ""
	}
	value = "/" + strings.Trim(value, "/")
	if value == "/" {
		return ""
	}
	return value
}

func isProduction(appEnv string) bool {
	switch strings.ToLower(strings.TrimSpace(appEnv)) {
	case "prod", "production":
		return true
	default:
		return false
	}
}

func strongSecret(value string) bool {
	return len(strings.TrimSpace(value)) >= minSecretLength
}

func validateHTTPURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("missing host")
	}
	return nil
}

func validateHTTPOrigin(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme")
	}
	if u.Host == "" {
		return fmt.Errorf("missing host")
	}
	if u.Path != "" && u.Path != "/" {
		return fmt.Errorf("origin must not include path")
	}
	if u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return fmt.Errorf("origin must not include query, fragment, or user info")
	}
	return nil
}

func validateBasePath(path string) error {
	if path == "" {
		return nil
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("missing leading slash")
	}
	if path != strings.TrimRight(path, "/") {
		return fmt.Errorf("must not end with slash")
	}
	if strings.ContainsAny(path, " ?#") {
		return fmt.Errorf("contains invalid characters")
	}
	return nil
}
