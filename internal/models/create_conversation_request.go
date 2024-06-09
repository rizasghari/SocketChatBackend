package models

type CreateConversationRequestBody struct {
	Users []uint `json:"users"`
	Type  string `json:"type"`
	Name  string `json:"name"`
}