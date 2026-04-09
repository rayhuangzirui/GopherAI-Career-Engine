package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/repository"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/service/analyzer"
)

var errTaskMarkedFailed = fmt.Errorf("task marked failed")

type TaskConsumer struct {
	conn             *amqp.Connection
	channel          *amqp.Channel
	queue            amqp.Queue
	taskRepo         *repository.TaskRepository
	processedKeyRepo *repository.ProcessedKeyRepository
	analyzer         analyzer.Analyzer
}

func NewTaskConsumer(
	rabbitMQURL string,
	taskRepo *repository.TaskRepository,
	processedKeyRepo *repository.ProcessedKeyRepository,
	analyzer analyzer.Analyzer,
) (*TaskConsumer, error) {
	conn, ch, q, err := setupQueue(rabbitMQURL, TaskQueueName)
	if err != nil {
		return nil, err
	}

	return &TaskConsumer{
		conn:             conn,
		channel:          ch,
		queue:            q,
		taskRepo:         taskRepo,
		processedKeyRepo: processedKeyRepo,
		analyzer:         analyzer,
	}, nil
}

func (c *TaskConsumer) ConsumeTasks(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		c.queue.Name,
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

	log.Printf("worker consuming queue: %s\n", c.queue.Name)

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
		return fmt.Errorf("mark processing task %d: %w", task.ID, err)
	}

	resultBytes, err := c.executeTask(ctx, task)
	if err != nil {
		if errors.Is(err, errTaskMarkedFailed) {
			return nil
		}
		return err
	}

	if err := c.taskRepo.MarkCompleted(ctx, task.ID, string(resultBytes)); err != nil {
		return fmt.Errorf("mark completed task %d: %w", task.ID, err)
	}

	if err := c.processedKeyRepo.Create(ctx, taskMessage.MessageKey); err != nil {
		return fmt.Errorf("create processed key %q: %w", taskMessage.MessageKey, err)
	}

	log.Printf("task %d completed\n", task.ID)
	return nil
}

func (c *TaskConsumer) executeTask(ctx context.Context, task *model.Task) ([]byte, error) {
	switch task.TaskType {
	case model.TaskTypeResumeAnalysis:
		return c.handleResumeAnalysis(ctx, task)
	case model.TaskTypeResumeJDMatch:
		return c.handleResumeJDMatch(ctx, task)
	default:
		return nil, c.failTask(ctx, task.ID, "unsupported task type", nil)
	}
}

func (c *TaskConsumer) handleResumeAnalysis(ctx context.Context, task *model.Task) ([]byte, error) {
	var input model.ResumeAnalysisInput
	if err := json.Unmarshal([]byte(task.InputPayload), &input); err != nil {
		return nil, c.failTask(ctx, task.ID, "failed to parse input payload", err)
	}

	result, err := c.analyzer.AnalyzeResume(input)
	if err != nil {
		return nil, c.failTask(ctx, task.ID, err.Error(), err)
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, c.failTask(ctx, task.ID, "failed to marshal result payload", err)
	}

	return resultBytes, nil
}

func (c *TaskConsumer) handleResumeJDMatch(ctx context.Context, task *model.Task) ([]byte, error) {
	var input model.ResumeJDMatchInput
	if err := json.Unmarshal([]byte(task.InputPayload), &input); err != nil {
		return nil, c.failTask(ctx, task.ID, "failed to parse input payload", err)
	}

	result, err := c.analyzer.MatchResumeJD(input)
	if err != nil {
		return nil, c.failTask(ctx, task.ID, err.Error(), err)
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, c.failTask(ctx, task.ID, "failed to marshal result payload", err)
	}

	return resultBytes, nil
}

func (c *TaskConsumer) failTask(ctx context.Context, taskID int64, message string, cause error) error {
	if err := c.taskRepo.MarkFailed(ctx, taskID, message); err != nil {
		if cause != nil {
			return fmt.Errorf("%s: %v; mark faild: %w", message, cause, err)
		}
		return fmt.Errorf("%s: %v", message, err)
	}
	return errTaskMarkedFailed
}

func (c *TaskConsumer) Close() error {
	return closeAMQPResources(c.channel, c.conn)
}
