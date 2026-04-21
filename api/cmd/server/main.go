package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/config"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/handler"
	mysqlinfra "github.com/rayhuangzirui/GopherAI-Career-Engine/internal/infra/mysql"
	redisinfra "github.com/rayhuangzirui/GopherAI-Career-Engine/internal/infra/redis"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/mq"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/repository"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/service/ratelimit"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/service/taskcache"
	"github.com/redis/go-redis/v9"
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

	redisClient, err := initRedisWithRetry(redisinfra.Config{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}, 10, 3*time.Second)
	if err != nil {
		log.Fatalf("init redis failed after retries: %v", err)
	}
	defer redisClient.Close()

	taskPublisher, err := mq.NewTaskPublisher(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("init rabbitmq failed: %v", err)
	}
	defer taskPublisher.Close()

	rateLimiter := ratelimit.New(redisClient)
	taskCache := taskcache.New(
		redisClient,
		time.Duration(cfg.TaskCacheTTLSeconds)*time.Second,
		time.Duration(cfg.TaskListCacheTTLSeconds)*time.Second,
		time.Duration(cfg.TaskResultCacheTTLSeconds)*time.Second,
	)

	r := gin.Default()
	r.Use(corsMiddleware())

	registerRoutes(r, cfg, db, taskPublisher, rateLimiter, taskCache)

	log.Printf("starting server on port %s in %s mode", cfg.Port, cfg.AppEnv)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("run server failed: %v", err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, Authorization, Origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
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

func initRedisWithRetry(cfg redisinfra.Config, maxAttempts int, delay time.Duration) (*redis.Client, error) {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client, err := redisinfra.New(cfg)
		if err == nil {
			log.Printf("redis connected on attempt %d/%d", attempt, maxAttempts)
			return client, nil
		}

		lastErr = err
		log.Printf("redis connect attempt %d/%d failed: %v", attempt, maxAttempts, err)

		if attempt < maxAttempts {
			time.Sleep(delay)
		}
	}

	return nil, lastErr
}

func registerRoutes(
	r *gin.Engine,
	cfg *config.Config,
	db *gorm.DB,
	taskPublisher *mq.TaskPublisher,
	rateLimiter *ratelimit.RateLimiter,
	taskCache *taskcache.TaskCache,
) {
	taskRepo := repository.NewTaskRepository(db)
	taskHandler := handler.NewTaskHandler(
		taskRepo,
		taskPublisher,
		rateLimiter,
		cfg.RateLimitPerMinute,
		taskCache,
	)

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

	r.POST("/tasks/resume-analysis", taskHandler.CreateResumeAnalysisTask)
	r.POST("/tasks/resume-jd-match", taskHandler.CreateResumeJDMatchTask)
	r.GET("/tasks", taskHandler.ListTasks)
	r.GET("/tasks/:id", taskHandler.GetTask)
	r.GET("/tasks/:id/result", taskHandler.GetTaskResult)
}
