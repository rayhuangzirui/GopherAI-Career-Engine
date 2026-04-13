package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	TaskQueueName = "career_tasks"
	TaskRetryQueueName = "career_tasks_retry"
)

type TaskPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	mainQueue   amqp.Queue
	retryQueue amqp.Queue
}

func NewTaskPublisher(rabbitURL string) (*TaskPublisher, error) {
	conn, ch, mainQ, retryQ, err := setupTaskQueues(rabbitURL)
	if err != nil {
		return nil, err
	}

	return &TaskPublisher{
		conn: conn,
		channel: ch,
		mainQueue: mainQ,
		retryQueue: retryQ,
	}, nil
}

func BuildTaskMessageKey(taskType string, taskID int64, attempt int) string {
	return fmt.Sprintf("%s:%d:attempt:%d", taskType, taskID, attempt)
}

func (p *TaskPublisher) PublishTask(ctx context.Context, taskID int64, taskType string, attempt int) error {
	msg := TaskMessage{
		TaskID:     taskID,
		TaskType: 	taskType,
		Attempt: 	attempt,
		MessageKey: BuildTaskMessageKey(taskType, taskID, attempt),
	}

	body, err := json.Marshal(msg)

	if err != nil {
		return fmt.Errorf("marshal task message: %w", err)
	}

	err = p.channel.PublishWithContext(
		ctx,
		"",
		p.mainQueue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish task message: %w", err)
	}

	return nil
}

func (p *TaskPublisher) PublishRetryTask(ctx context.Context, taskID int64, taskType string, attempt int, delay time.Duration) error {
	msg := TaskMessage{
		TaskID:     taskID,
		TaskType: 	taskType,
		Attempt: 	attempt,
		MessageKey: BuildTaskMessageKey(taskType, taskID, attempt),
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal retry task message: %w", err)
	}

	err = p.channel.PublishWithContext(
		ctx,
			"",
			p.retryQueue.Name,
			false,
			false,
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Body:         body,
				Expiration:   fmt.Sprintf("%d", delay.Milliseconds()), // delay in milliseconds
			},
	)
	if err != nil {
		return fmt.Errorf("publish retry task message: %w", err)
	}

	return nil
}

func (p *TaskPublisher) Close() error {
	return closeAMQPResources(p.channel, p.conn)
}
