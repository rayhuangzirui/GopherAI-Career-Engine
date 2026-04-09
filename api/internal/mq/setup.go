package mq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func setupQueue (rabbitMQURL string, queueName string) (*amqp.Connection, *amqp.Channel, amqp.Queue, error) {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, nil, amqp.Queue{}, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, amqp.Queue{}, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, amqp.Queue{}, fmt.Errorf("declare rabbitmq queue: %w", err)
	}

	return conn, ch, q, nil
}
