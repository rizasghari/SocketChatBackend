package models

import "gorm.io/gorm"

// Message represents a message sent in a conversation
type Message struct {
	gorm.Model
	ConversationID uint         `json:"conversation_id"`
	Conversation   Conversation `json:"-"`
	SenderID       uint         `json:"sender_id"`
	Content        string       `gorm:"not null" json:"content"`
}
