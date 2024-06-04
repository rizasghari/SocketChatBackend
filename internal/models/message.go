package models

type Message struct {
	ReceiverID uint   `json:"receiver_id"`
	Content    string `json:"content"`
}
