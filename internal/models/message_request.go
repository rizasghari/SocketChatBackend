package models

type MessageRequest struct {
	ConversationID uint   `json:"conversation_id"`
	Content        string `json:"content"`
}