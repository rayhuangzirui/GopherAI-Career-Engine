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
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	q, err := channel.QueueDeclare(
		TaskQueueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("declare rabbitmq queue: %w", err)
	}

	return &TaskPublisher{conn: conn, channel: channel, queue: q}, nil
}

func BuildTaskMessageKey(taskID int64) string {
	return fmt.Sprintf("resume_analysis:%d", taskID)
}

func (p *TaskPublisher) PublishTask(ctx context.Context, taskID int64) error {
	msg := TaskMessage{
		TaskID:     taskID,
		MessageKey: BuildTaskMessageKey(taskID),
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
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
