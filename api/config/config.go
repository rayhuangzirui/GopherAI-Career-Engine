package config

import (
	"fmt"
	"os"
	"strconv"
	"sync"
)

type Config struct {
	AppEnv      string
	Port        string
	MySQLDSN    string
	RedisAddr   string
	RedisPassword string
	RedisDB     int
	RabbitMQURL string
	JWTSecret   string

	AnalyzerMode       string
	LLMProvider        string
	LLMBaseURL         string
	LLMAPIKey          string
	LLMModel           string
	LLMTimeoutSeconds  int
	LLMTemperature     float64
	LLMMaxInputChars   int
	LLMMaxOutputTokens int

	RateLimitPerMinute int
}

var (
	cfg  *Config
	once sync.Once
)

func Load() *Config {
	once.Do(func() {
		cfg = &Config{
			AppEnv:      getEnv("APP_ENV", "dev"),
			Port:        getEnv("PORT", "8080"),
			MySQLDSN:    mustGetEnv("MYSQL_DSN"),
			RedisAddr:   mustGetEnv("REDIS_ADDR"),
			RedisPassword: getEnv("REDIS_PASSWORD", ""),
			RedisDB:     getEnvInt("REDIS_DB", 0),
			RabbitMQURL: mustGetEnv("RABBITMQ_URL"),
			JWTSecret:   mustGetEnv("JWT_SECRET"),

			AnalyzerMode:       getEnv("ANALYZER_MODE", "rules"),
			LLMProvider:        getEnv("LLM_PROVIDER", "dashscope"),
			LLMBaseURL:         getEnv("LLM_BASE_URL", "http://dashscope-us.aliyuncs.com/compatible-mode/v1"),
			LLMAPIKey:          getEnv("LLM_API_KEY", getEnv("DASHSCOPE_API_KEY", "")),
			LLMModel:           getEnv("LLM_MODEL", "qwen-plus"),
			LLMTimeoutSeconds:  getEnvInt("LLM_TIMEOUT_SECONDS", 20),
			LLMTemperature:     getEnvFloat("LLM_TEMPERATURE", 0.2),
			LLMMaxInputChars:   getEnvInt("LLM_MAX_INPUT_CHARS", 8000),
			LLMMaxOutputTokens: getEnvInt("LLM_MAX_OUTPUT_TOKENS", 800),

			RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 10),
		}
	})
	return cfg
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("missing required environment variable: %s", key))
	}
	return value
}

func getEnv(key, def string) string {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	return value
}

func getEnvInt(key string, def int) int {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("invalid int environment variable %s=%q", key, value))
	}
	return parsed
}

func getEnvFloat(key string, def float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(fmt.Sprintf("invalid float environment variable %s=%q", key, value))
	}
	return parsed
}
