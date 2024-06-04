package models

import "gorm.io/gorm"

// Conversation represents a group or private conversation
type Conversation struct {
	gorm.Model
	Type     string    `gorm:"not null" json:"type"`
	Name     *string   `json:"name"`
	Members  []User    `gorm:"many2many:conversation_members;"`
	Messages []Message `json:"messages"`
}
