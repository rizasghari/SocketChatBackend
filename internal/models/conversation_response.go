package models

type ConversationResponse struct {
	ID      uint           `json:"id"`
	Type    string         `json:"type"`
	Name    *string        `json:"name"`
	Members []UserResponse `json:"members"`
}
