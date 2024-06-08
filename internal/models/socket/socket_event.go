package models

import (
	"encoding/json"
)

type SocketEvent struct {
	Event          string `json:"event"`
	Payload        json.RawMessage    `json:"payload"`
	ConversationID uint   `json:"conversation_id"`
}
