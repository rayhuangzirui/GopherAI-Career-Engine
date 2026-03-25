package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/config"
	mysqlinfra "github.com/rayhuangzirui/GopherAI-Career-Engine/internal/infra/mysql"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	db, err := initMySQLWithRetry(mysqlinfra.Config{
		DSN:          cfg.MySQLDSN,
		MaxIdleConns: 10,
		MaxOpenConns: 20,
		MaxLifetime:  30 * time.Minute,
	}, 10, 3*time.Second)

	if err != nil {
		log.Fatalf("init mysql failed after retries: %v", err)
	}

	r := gin.Default()

	registerRoutes(r, cfg, db)

	log.Printf("starting server on port %s in %s mode", cfg.Port, cfg.AppEnv)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("run server failed: %v", err)
	}
}

func initMySQLWithRetry(cfg mysqlinfra.Config, maxAttempts int, delay time.Duration) (*gorm.DB, error) {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		db, err := mysqlinfra.New(cfg)
		if err == nil {
			log.Printf("mysql connected on attempt %d/%d", attempt, maxAttempts)
			return db, nil
		}

		lastErr = err
		log.Printf("mysql connect attempt %d/%d failed: %v", attempt, maxAttempts, err)

		if attempt < maxAttempts {
			time.Sleep(delay)
		}
	}
	return nil, lastErr
}

func registerRoutes(r *gin.Engine, cfg *config.Config, db *gorm.DB) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":       true,
			"env":      cfg.AppEnv,
			"mysql":    db != nil,
			"redis":    cfg.RedisAddr != "",
			"rabbitmq": cfg.RabbitMQURL != "",
		})
	})

	r.GET("/debug/db", func(c *gin.Context) {
		var usersCount int64

		if err := db.Table("users").Count(&usersCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"ok":    false,
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":         true,
			"usersCount": usersCount,
		})
	})
}
