package models

type IsTypingPayload struct {
	TypingStatus bool `json:"typing_status"`
	UserID       uint `json:"user_id"`
}
