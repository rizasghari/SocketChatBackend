package models

type ConversationResponse struct {
	ID          uint            `json:"id"`
	Type        string          `json:"type"`
	Members     []*UserResponse `json:"members"`
	LastMessage *Message        `json:"last_message"`
	Unread      int             `json:"unread"`
	Whiteboard  *Whiteboard     `json:"whiteboard"`
}
