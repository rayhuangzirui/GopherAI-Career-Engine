package main

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	if os.Getenv("APP_ENV") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		_ = start // 先占位，后面你加 structured log / request_id
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":    true,
			"env":   os.Getenv("APP_ENV"),
			"mysql": os.Getenv("MYSQL_DSN") != "",
			"redis": os.Getenv("REDIS_ADDR") != "",
			"mq":    os.Getenv("RABBITMQ_URL") != "",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	_ = r.Run(":" + port)
	_ = r.SetTrustedProxies(nil)
}
