package mq

import amqp "github.com/rabbitmq/amqp091-go"

func closeAMQPResources(ch *amqp.Channel, conn *amqp.Connection) error {
	if ch != nil {
		_ = ch.Close()
	}

	if conn != nil {
		return conn.Close()
	}
	return nil
}
