package models

import "time"

// ChatMessage 聊天消息
type ChatMessage struct {
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	IsBot     bool      `json:"is_bot"`
}
