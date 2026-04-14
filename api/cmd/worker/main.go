package main

import (
	"context"
	"log"
	"time"

	"github.com/rayhuangzirui/GopherAI-Career-Engine/config"
	mysqlinfra "github.com/rayhuangzirui/GopherAI-Career-Engine/internal/infra/mysql"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/mq"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/repository"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/service/analyzer"
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

	taskRepo := repository.NewTaskRepository(db)
	processedKeyRepo := repository.NewProcessedKeyRepository(db)
	//mockAnalyzer := analyzer.NewMockAnalyzer()
	ruleAnalyzer := analyzer.NewRulesAnalyzer()

	consumer, err := initTaskConsumerWithRetry(
		cfg.RabbitMQURL,
		taskRepo,
		processedKeyRepo,
		//mockAnalyzer,
		ruleAnalyzer,
		mq.RetryConfig{
			MaxRetries: 3,
		},
		10,
		3*time.Second,
	)
	if err != nil {
		log.Fatalf("init task consumer failed: %v", err)
	}
	defer consumer.Close()

	log.Printf("worker started in %s mode, consuming queue %s", cfg.AppEnv, mq.TaskQueueName)
	if err := consumer.ConsumeTasks(context.Background()); err != nil {
		log.Fatalf("consumer stopped with error: %v", err)
	}
}

func initTaskConsumerWithRetry(
	rabbitmqURL string,
	taskRepo *repository.TaskRepository,
	processedKeyRepo *repository.ProcessedKeyRepository,
	analyzer analyzer.Analyzer,
	retryConfig mq.RetryConfig,
	maxAttempts int,
	delay time.Duration,
) (*mq.TaskConsumer, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		consumer, err := mq.NewTaskConsumer(rabbitmqURL, taskRepo, processedKeyRepo, analyzer, retryConfig)
		if err == nil {
			log.Printf("rabbitmq task consumer connected on attempt %d/%d", attempt, maxAttempts)
			return consumer, nil
		}

		lastErr = err
		log.Printf("rabbitmq task consumer connect attempt %d/%d failed: %v", attempt, maxAttempts, err)

		if attempt < maxAttempts {
			time.Sleep(delay)
		}
	}

	return nil, lastErr
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
