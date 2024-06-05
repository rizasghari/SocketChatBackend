package models

type TempSocketMessage struct {
	ReceiverID uint   `json:"receiver_id"`
	Content    string `json:"content"`
}
