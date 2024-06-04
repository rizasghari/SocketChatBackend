package models

import (
	"gorm.io/gorm"
	"time"
)

// ConversationMember represents the mapping of users to conversations
type ConversationMember struct {
	gorm.Model
	ConversationID uint      `json:"conversation_id"`
	UserID         uint      `json:"user_id"`
	JoinedAt       time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"joined_at"`
}
