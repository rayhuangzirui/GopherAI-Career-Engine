package config

import (
	"fmt"
	"os"
	"sync"
)

type Config struct {
	AppEnv 		string
	Port 		string
	MySQLDSN 	string
	RedisAddr 	string
	RabbitMQURL string
	JWTSecret 	string
}

var (
	cfg 		*Config
	once 		sync.Once
)

func Load() *Config {
	once.Do(func() {
		cfg = &Config{
			AppEnv: 		getEnv("APP_ENV", "dev"),
			Port: 			getEnv("PORT", "8080"),
			MySQLDSN: 		mustGetEnv("MYSQL_DSN"),
			RedisAddr: 		mustGetEnv("REDIS_ADDR"),
			RabbitMQURL: 	mustGetEnv("RABBITMQ_URL"),
			JWTSecret: 		mustGetEnv("JWT_SECRET"),
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
