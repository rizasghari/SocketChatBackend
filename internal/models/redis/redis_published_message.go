package models

type RedisPublishedMessage struct {
	Event          string `json:"event"`
	ConversationID uint   `json:"conversation_id"`
	Payload        any    `json:"payload"`
}
