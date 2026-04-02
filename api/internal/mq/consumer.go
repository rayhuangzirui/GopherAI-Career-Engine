package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/model"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/repository"
	"github.com/rayhuangzirui/GopherAI-Career-Engine/internal/service/analyzer"
)

type TaskConsumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	queue    amqp.Queue
	taskRepo *repository.TaskRepository
	analyzer analyzer.Analyzer
}

func NewTaskConsumer(
	rabbitMQURL string,
	taskRepo *repository.TaskRepository,
	analyzer analyzer.Analyzer,
) (*TaskConsumer, error) {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	q, err := ch.QueueDeclare(
		TaskQueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare rabbitmq queue: %w", err)
	}

	return &TaskConsumer{
		conn:     conn,
		channel:  ch,
		queue:    q,
		taskRepo: taskRepo,
		analyzer: analyzer,
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

	task, err := c.taskRepo.GetTask(ctx, taskMessage.TaskID)
	if err != nil {
		return fmt.Errorf("get task %d: %w", taskMessage.TaskID, err)
	}

	if task.TaskType != model.TaskTypeResumeAnalysis {
		failErr := c.taskRepo.MarkFailed(ctx, task.ID, "unsupported task type")
		if failErr != nil {
			return fmt.Errorf("unsupported task type and mark failed: %w", failErr)
		}
		return nil
	}

	if err := c.taskRepo.MarkProcessing(ctx, task.ID); err != nil {
		return fmt.Errorf("mark processing task %d: %w", task.ID, err)
	}

	var input model.ResumeAnalysisInput
	if err := json.Unmarshal([]byte(task.InputPayload), &input); err != nil {
		failErr := c.taskRepo.MarkFailed(ctx, task.ID, "failed to parse input payload")
		if failErr != nil {
			return fmt.Errorf("parse input payload: %v; mark failed: %w", err, failErr)
		}

		return nil
	}

	result, err := c.analyzer.AnalyzeResume(input)
	if err != nil {
		failErr := c.taskRepo.MarkFailed(ctx, task.ID, err.Error())
		if failErr != nil {
			return fmt.Errorf("analyze task: %v; mark failed: %w", err, failErr)
		}
		return nil
	}
	resultBytes, err := json.Marshal(result)
	if err != nil {
		failErr := c.taskRepo.MarkFailed(ctx, task.ID, "failed to marshal result payload")
		if failErr != nil {
			return fmt.Errorf("marshal result payload: %v; mark failed: %w", err, failErr)
		}
		return nil
	}

	if err := c.taskRepo.MarkCompleted(ctx, task.ID, string(resultBytes)); err != nil {
		return fmt.Errorf("mark completed task %d: %w", task.ID, err)
	}

	log.Printf("task %d completed\n", task.ID)
	return nil
}

func (c *TaskConsumer) Close() error {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
