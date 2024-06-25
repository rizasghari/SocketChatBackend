package whiteboard

import "gorm.io/gorm"

type Whiteboard struct {
	gorm.Model
	ConversationID uint    `json:"conversation_id"`
	Drawns         []Drawn `json:"drawns" gorm:"many2many:drawns;"`
}
