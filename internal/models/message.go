package models

import (
	"time"

	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	ConversationID uint         `json:"conversation_id"`
	Conversation   Conversation `json:"-"`
	SenderID       uint         `json:"sender_id"`
	Content        string       `gorm:"not null" json:"content"`
	SeenAt         *time.Time   `json:"seen_at"`
}
