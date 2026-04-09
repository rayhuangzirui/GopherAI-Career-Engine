package mq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const TaskQueueName = "career_tasks"

type TaskPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

func NewTaskPublisher(rabbitURL string) (*TaskPublisher, error) {
	conn, ch, q, err := setupQueue(rabbitURL, TaskQueueName)
	if err != nil {
		return nil, err
	}

	return &TaskPublisher{conn: conn, channel: ch, queue: q}, nil
}

func BuildTaskMessageKey(taskType string, taskID int64) string {
	return fmt.Sprintf("%s:%d", taskType, taskID)
}

func (p *TaskPublisher) PublishTask(ctx context.Context, taskID int64, taskType string) error {
	msg := TaskMessage{
		TaskID:     taskID,
		MessageKey: BuildTaskMessageKey(taskType, taskID),
	}

	body, err := json.Marshal(msg)

	if err != nil {
		return fmt.Errorf("marshal task message: %w", err)
	}

	err = p.channel.PublishWithContext(
		ctx,
		"",
		p.queue.Name,
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

func (p *TaskPublisher) Close() error {
	return closeAMQPResources(p.channel, p.conn)
}
