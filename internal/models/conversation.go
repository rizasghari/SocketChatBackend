package models

import (
	"gorm.io/gorm"
)

type Conversation struct {
	gorm.Model
	Type       string      `gorm:"not null" json:"type"`
	Members    []User      `gorm:"many2many:conversation_members;"`
	Whiteboard *Whiteboard `json:"whiteboard"`
	Messages   []Message   `json:"messages"`
}

func (conversation *Conversation) ToConversationResponse(lastMessage *Message, unread int) ConversationResponse {
	members := []*UserResponse{}
	for _, member := range conversation.Members {
		members = append(members, member.ToUserResponse())
	}
	return ConversationResponse{
		ID:          conversation.ID,
		Type:        conversation.Type,
		Members:     members,
		LastMessage: lastMessage,
		Unread:      unread,
		Whiteboard:  conversation.Whiteboard,
	}
}
