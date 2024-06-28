package models

import "gorm.io/gorm"

type Whiteboard struct {
	gorm.Model
	ConversationID uint    `json:"conversation_id"`
	Drawns         []Drawn `json:"drawns"`
	Creator        uint    `json:"creator_user_id" gorm:"not null;column:creator_user_id"`
}