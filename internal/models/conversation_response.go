package models

type ConversationResponse struct {
	ID      uint           `json:"id"`
	Type    string         `json:"type"`
	Members []*UserResponse `json:"members"`
}
