package mq

type TaskMessage struct {
	TaskID     int64  `json:"task_id"`
	MessageKey string `json:"message_key"`
}
