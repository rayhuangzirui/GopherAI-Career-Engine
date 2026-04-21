package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/repository"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/service/analyzer"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/storage"
)

type TaskExecError struct {
	Message   string
	Cause     error
	Retryable bool
}

func (e *TaskExecError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

type RetryConfig struct {
	MaxRetries int
}

func (c RetryConfig) DelayForAttemp(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 5 * time.Second
	case 2:
		return 10 * time.Second
	case 3:
		return 30 * time.Second
	default:
		return 60 * time.Second
	}
}

type TaskConsumer struct {
	conn             *amqp.Connection
	channel          *amqp.Channel
	mainQueue        amqp.Queue
	retryQueue       amqp.Queue
	taskRepo         *repository.TaskRepository
	processedKeyRepo *repository.ProcessedKeyRepository
	analyzer         analyzer.Analyzer
	artifactStorage  storage.Storage
	publisher        *TaskPublisher
	retryConfig      RetryConfig
}

func NewTaskConsumer(
	rabbitMQURL string,
	taskRepo *repository.TaskRepository,
	processedKeyRepo *repository.ProcessedKeyRepository,
	analyzer analyzer.Analyzer,
	artifactStorage storage.Storage,
	retryConfig RetryConfig,
) (*TaskConsumer, error) {
	conn, ch, mainQ, retryQ, err := setupTaskQueues(rabbitMQURL)
	if err != nil {
		return nil, err
	}

	publisher := &TaskPublisher{conn: conn, channel: ch, mainQueue: mainQ, retryQueue: retryQ}

	return &TaskConsumer{
		conn:             conn,
		channel:          ch,
		mainQueue:        mainQ,
		retryQueue:       retryQ,
		taskRepo:         taskRepo,
		processedKeyRepo: processedKeyRepo,
		analyzer:         analyzer,
		artifactStorage:  artifactStorage,
		publisher:        publisher,
		retryConfig:      retryConfig,
	}, nil
}

func (c *TaskConsumer) ConsumeTasks(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		c.mainQueue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("register rabbitmq consumer: %w", err)
	}

	log.Printf("worker consuming queue: %s\n", c.mainQueue.Name)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("rabbitmq channel closed")
			}

			if err := c.handleMessage(ctx, msg); err != nil {
				log.Printf("handle message failed: %v\n", err)
				if nackErr := msg.Nack(false, false); nackErr != nil {
					log.Printf("nack message failed: %v\n", nackErr)
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				log.Printf("ack message failed: %v\n", err)
			}
		}
	}
}

func (c *TaskConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	var taskMessage TaskMessage
	if err := json.Unmarshal(msg.Body, &taskMessage); err != nil {
		return fmt.Errorf("unmarshal task message: %w", err)
	}

	if taskMessage.MessageKey == "" {
		return fmt.Errorf("message key is empty")
	}

	exists, err := c.processedKeyRepo.Exists(ctx, taskMessage.MessageKey)
	if err != nil {
		return fmt.Errorf("check processed key %q: %w", taskMessage.MessageKey, err)
	}

	if exists {
		log.Printf("skip already processed message: key=%s task_id=%d", taskMessage.MessageKey, taskMessage.TaskID)
		return nil
	}

	task, err := c.taskRepo.GetTask(ctx, taskMessage.TaskID)
	if err != nil {
		return fmt.Errorf("get task %d: %w", taskMessage.TaskID, err)
	}

	if err := c.taskRepo.MarkProcessing(ctx, task.ID); err != nil {
		if errors.Is(err, repository.ErrInvalidStateTransition) {
			log.Printf("skip task due to invalid transition to processing: key=%s task_id=%d", taskMessage.MessageKey, taskMessage.TaskID)
			return nil
		}
		return fmt.Errorf("mark processing task %d: %w", task.ID, err)
	}

	resultBytes, execErr := c.executeTask(ctx, task)
	if execErr == nil {
		resultPayload := string(resultBytes)

		if err := c.taskRepo.MarkCompleted(ctx, task.ID, resultPayload); err != nil {
			return fmt.Errorf("mark completed task %d: %w", task.ID, err)
		}

		if c.artifactStorage != nil {
			artifactKey := storage.BuildTaskResultArtifactKey(task.ID)

			if err := c.artifactStorage.Put(
				ctx,
				artifactKey,
				"application/json",
				resultBytes,
			); err != nil {
				log.Printf(
					"artifact_put_failed task_id=%d storage=%s err=%v",
					task.ID,
					c.artifactStorage.Name(),
					err,
				)
			} else {
				if err := c.taskRepo.UpdateArtifactLocation(
					ctx,
					task.ID,
					c.artifactStorage.Name(),
					artifactKey,
				); err != nil {
					log.Printf(
						"artifact_metadata_update_failed task_id=%d storage=%s key=%s err=%v",
						task.ID,
						c.artifactStorage.Name(),
						artifactKey,
						err,
					)
				} else {
					log.Printf(
						"artifact_put_success task_id=%d storage=%s key=%s",
						task.ID,
						c.artifactStorage.Name(),
						artifactKey,
					)
				}
			}
		}

		if err := c.processedKeyRepo.Create(ctx, taskMessage.MessageKey); err != nil {
			return fmt.Errorf("create processed key %q: %w", taskMessage.MessageKey, err)
		}

		log.Printf("task %d completed", task.ID)
		return nil
	}

	var taskErr *TaskExecError
	ok := errors.As(execErr, &taskErr)
	if !ok {
		return fmt.Errorf("execute task %d: unexpected task execution type: %w", task.ID, execErr)
	}

	if !taskErr.Retryable {
		if err := c.taskRepo.MarkPermanentlyFailed(ctx, task.ID, taskErr.Error()); err != nil {
			return fmt.Errorf("mark permanently failed task %d: %w", task.ID, err)
		}

		if err := c.processedKeyRepo.Create(ctx, taskMessage.MessageKey); err != nil {
			return fmt.Errorf("create processed key %q: %w", taskMessage.MessageKey, err)
		}
		log.Printf("task %d permanently failed (non-retryable): %s", task.ID, taskErr.Error())
		return nil
	}

	// retryable
	if task.RetryCount >= c.retryConfig.MaxRetries {
		if err := c.taskRepo.MarkPermanentlyFailed(ctx, task.ID, taskErr.Error()); err != nil {
			return fmt.Errorf("mark permanently failed task %d: %w", task.ID, err)
		}

		if err := c.processedKeyRepo.Create(ctx, taskMessage.MessageKey); err != nil {
			return fmt.Errorf("create processed key %q: %w", taskMessage.MessageKey, err)
		}
		log.Printf("task %d permanently failed (max retries reached): %s", task.ID, taskErr.Error())
		return nil
	}

	nextAttempt := taskMessage.Attempt + 1
	delay := c.retryConfig.DelayForAttemp(nextAttempt)

	if err := c.taskRepo.MarkRetrying(ctx, task.ID, taskErr.Error()); err != nil {
		return fmt.Errorf("mark retrying task %d: %w", task.ID, err)
	}

	if err := c.publisher.PublishRetryTask(ctx, task.ID, task.TaskType, nextAttempt, delay); err != nil {
		return fmt.Errorf("publish retry task %d attempt %d after %s: %w", task.ID, nextAttempt, delay, err)
	}

	//if err := c.publisher.PublishTask(ctx, task.ID, task.TaskType, nextAttempt); err != nil {
	//	return fmt.Errorf("republish retry task %d attempt %d: %w", task.ID, nextAttempt, err)
	//}

	//if err := c.taskRepo.RequeueFromRetrying(ctx, task.ID); err != nil {
	//	return fmt.Errorf("requeue retrying task %d: %w", task.ID, err)
	//}

	if err := c.processedKeyRepo.Create(ctx, taskMessage.MessageKey); err != nil {
		return fmt.Errorf("create processed key %q: %w", taskMessage.MessageKey, err)
	}

	log.Printf("task %d scheduled retry attempt %d after %s", task.ID, nextAttempt, delay)
	return nil
}

func (c *TaskConsumer) executeTask(ctx context.Context, task *model.Task) ([]byte, error) {
	switch task.TaskType {
	case model.TaskTypeResumeAnalysis:
		return c.handleResumeAnalysis(ctx, task)
	case model.TaskTypeResumeJDMatch:
		return c.handleResumeJDMatch(ctx, task)
	default:
		return nil, &TaskExecError{
			Message:   "unsupported task type",
			Cause:     nil,
			Retryable: false,
		}
	}
}

func (c *TaskConsumer) handleResumeAnalysis(ctx context.Context, task *model.Task) ([]byte, error) {
	var input model.ResumeAnalysisInput
	if err := json.Unmarshal([]byte(task.InputPayload), &input); err != nil {
		return nil, &TaskExecError{
			Message:   "failed to parse input payload",
			Cause:     err,
			Retryable: false,
		}
	}

	result, err := c.analyzer.AnalyzeResume(input)
	if err != nil {
		return nil, &TaskExecError{
			Message:   "resume analysis failed",
			Cause:     err,
			Retryable: true,
		}
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, &TaskExecError{
			Message:   "failed to marshal result payload",
			Cause:     err,
			Retryable: false,
		}
	}

	return resultBytes, nil
}

func (c *TaskConsumer) handleResumeJDMatch(ctx context.Context, task *model.Task) ([]byte, error) {
	var input model.ResumeJDMatchInput
	if err := json.Unmarshal([]byte(task.InputPayload), &input); err != nil {
		return nil, &TaskExecError{
			Message:   "failed to parse input payload",
			Cause:     err,
			Retryable: false,
		}
	}

	result, err := c.analyzer.MatchResumeJD(input)
	if err != nil {
		return nil, &TaskExecError{
			Message:   "resume JD match analysis failed",
			Cause:     err,
			Retryable: true,
		}
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, &TaskExecError{
			Message:   "failed to marshal result payload",
			Cause:     err,
			Retryable: false,
		}
	}

	return resultBytes, nil
}

func isRetryableError(err error) bool {
	// Unretryable errors:
	// failed to parse input payload
	// failed to marshal result payload
	// unsupported task type

	// Retryable errors:
	// analyzer returns runtime error
	// analyzer returns analysis error
	// LLM/external API call fails
	return true
}

func (c *TaskConsumer) Close() error {
	return closeAMQPResources(c.channel, c.conn)
}
