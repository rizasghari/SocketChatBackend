package models

type ConversationResponse struct {
	Type    string         `json:"type"`
	Name    *string        `json:"name"`
	Members []UserResponse `json:"members"`
}
