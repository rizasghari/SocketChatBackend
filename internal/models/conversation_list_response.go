package models

type ConversationListResponse struct {
	Conversations []Conversation `json:"conversations"`
	Page          int            `json:"page"`
	Size          int            `json:"size"`
	Total         int64          `json:"total"`
}
