package mq

type TaskMessage struct {
	TaskID     int64  `json:"task_id"`
	TaskType   string `json:"task_type"`
	Attempt	   int    `json:"attempt"`
	MessageKey string `json:"message_key"`
}
