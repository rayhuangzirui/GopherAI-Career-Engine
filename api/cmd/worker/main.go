package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rayhuangzirui/GopherAI-Career-Engine/config"
	mysqlinfra "github.com/rayhuangzirui/GopherAI-Career-Engine/internal/infra/mysql"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/mq"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/repository"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/service/analyzer"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/storage"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/storage/localstorage"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/storage/s3storage"
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
	//ruleAnalyzer := analyzer.NewRulesAnalyzer()
	taskAnalyzer, err := buildAnalyzer(cfg)
	artifactStorage, err := buildStorage(cfg)
	if err != nil {
		log.Fatalf("build storage failed: %v", err)
	}

	if err != nil {
		log.Fatalf("build analyzer failed: %v", err)
	}

	consumer, err := initTaskConsumerWithRetry(
		cfg.RabbitMQURL,
		taskRepo,
		processedKeyRepo,
		//mockAnalyzer,
		taskAnalyzer,
		artifactStorage,
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

	log.Printf("worker started in %s mode, consuming queue %s, analyzer_mode=%s, llm_provider=%s, llm_model=%s", cfg.AppEnv, mq.TaskQueueName, cfg.AnalyzerMode, cfg.LLMProvider, cfg.LLMModel)
	if err := consumer.ConsumeTasks(context.Background()); err != nil {
		log.Fatalf("consumer stopped with error: %v", err)
	}
}

func buildAnalyzer(cfg *config.Config) (analyzer.Analyzer, error) {
	switch cfg.AnalyzerMode {
	case "rules":
		return analyzer.NewRulesAnalyzer(), nil
	case "llm":
		if cfg.LLMAPIKey == "" {
			return nil, fmt.Errorf("ANALYZER_MODE=llm but no LLM_API_KEY or DASHSCOPE_API_KEY is set")
		}
		fallback := analyzer.NewRulesAnalyzer()
		return analyzer.NewLLMAnalyzer(cfg, fallback), nil
	default:
		return nil, fmt.Errorf("unknown analyzer mode: %s", cfg.AnalyzerMode)
	}
}

func initTaskConsumerWithRetry(
	rabbitmqURL string,
	taskRepo *repository.TaskRepository,
	processedKeyRepo *repository.ProcessedKeyRepository,
	analyzer analyzer.Analyzer,
	artifactStorage storage.Storage,
	retryConfig mq.RetryConfig,
	maxAttempts int,
	delay time.Duration,
) (*mq.TaskConsumer, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		consumer, err := mq.NewTaskConsumer(rabbitmqURL, taskRepo, processedKeyRepo, analyzer, artifactStorage, retryConfig)
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

func buildStorage(cfg *config.Config) (storage.Storage, error) {
	switch cfg.ArtifactStorageDriver {
	case "local":
		return localstorage.NewLocalStorage(cfg.ArtifactLocalBaseDir)
	case "s3":
		return s3storage.New(s3storage.Config{
			Region: cfg.AWSRegion,
			Bucket: cfg.S3Bucket,
			Endpoint: cfg.S3Endpoint,
			AccessKeyID: cfg.S3AccessKeyID,
			SecretAccessKey: cfg.S3SecretAccessKey,
			ForcePathStyle: cfg.S3ForcePathStyle,
		})
	default:
		return nil, fmt.Errorf("unknown artifact storage driver: %s", cfg.ArtifactStorageDriver)
	}
}