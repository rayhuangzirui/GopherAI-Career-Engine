package config

import (
	"fmt"
	"os"
	"strconv"
	"sync"
)

type Config struct {
	AppEnv        string
	Port          string
	MySQLDSN      string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RabbitMQURL   string
	JWTSecret     string

	AnalyzerMode       string
	LLMProvider        string
	LLMBaseURL         string
	LLMAPIKey          string
	LLMModel           string
	LLMTimeoutSeconds  int
	LLMTemperature     float64
	LLMMaxInputChars   int
	LLMMaxOutputTokens int

	RateLimitPerMinute        int
	TaskCacheTTLSeconds       int
	TaskListCacheTTLSeconds   int
	TaskResultCacheTTLSeconds int

	ArtifactStorageDriver string
	ArtifactLocalBaseDir  string

	AWSRegion   	string
	S3Bucket        string
	S3Endpoint 		string
	S3AccessKeyID 	string
	S3SecretAccessKey string
	S3ForcePathStyle bool
}

var (
	cfg  *Config
	once sync.Once
)

func Load() *Config {
	once.Do(func() {
		cfg = &Config{
			AppEnv:        getEnv("APP_ENV", "dev"),
			Port:          getEnv("PORT", "8080"),
			MySQLDSN:      mustGetEnv("MYSQL_DSN"),
			RedisAddr:     mustGetEnv("REDIS_ADDR"),
			RedisPassword: getEnv("REDIS_PASSWORD", ""),
			RedisDB:       getEnvInt("REDIS_DB", 0),
			RabbitMQURL:   mustGetEnv("RABBITMQ_URL"),
			JWTSecret:     mustGetEnv("JWT_SECRET"),

			AnalyzerMode:       getEnv("ANALYZER_MODE", "rules"),
			LLMProvider:        getEnv("LLM_PROVIDER", "dashscope"),
			LLMBaseURL:         getEnv("LLM_BASE_URL", "https://dashscope-us.aliyuncs.com/compatible-mode/v1"),
			LLMAPIKey:          getEnv("LLM_API_KEY", getEnv("DASHSCOPE_API_KEY", "")),
			LLMModel:           getEnv("LLM_MODEL", "qwen-plus"),
			LLMTimeoutSeconds:  getEnvInt("LLM_TIMEOUT_SECONDS", 20),
			LLMTemperature:     getEnvFloat("LLM_TEMPERATURE", 0.2),
			LLMMaxInputChars:   getEnvInt("LLM_MAX_INPUT_CHARS", 8000),
			LLMMaxOutputTokens: getEnvInt("LLM_MAX_OUTPUT_TOKENS", 800),

			RateLimitPerMinute:        getEnvInt("RATE_LIMIT_PER_MINUTE", 10),
			TaskCacheTTLSeconds:       getEnvInt("TASK_CACHE_TTL_SECONDS", 60),
			TaskListCacheTTLSeconds:   getEnvInt("TASK_LIST_CACHE_TTL_SECONDS", 10),
			TaskResultCacheTTLSeconds: getEnvInt("TASK_RESULT_CACHE_TTL_SECONDS", 300),

			ArtifactStorageDriver: getEnv("ARTIFACT_STORAGE_DRIVER", "local"),
			ArtifactLocalBaseDir: getEnv("ARTIFACT_LOCAL_BASE_DIR", "./data"),

			AWSRegion: getEnv("AWS_REGION", ""),
			S3Bucket: getEnv("S3_BUCKET", ""),
			S3Endpoint: getEnv("S3_ENDPOINT", ""),
			S3AccessKeyID: getEnv("S3_ACCESS_KEY_ID", ""),
			S3SecretAccessKey: getEnv("S3_SECRET_ACCESS_KEY", ""),
			S3ForcePathStyle: getEnvBool("S3_FORCE_PATH_STYLE", false),
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

func getEnvBool(key string, def bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return def
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		panic(fmt.Sprintf("invalid bool environment variable %s=%q", key, value))
	}
	return parsed
}