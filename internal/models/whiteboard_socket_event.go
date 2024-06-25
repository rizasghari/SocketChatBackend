package models

type WhiteboardSocketEvent struct {
	Event   string                  `json:"event"`
	Payload WhiteboardSocketPayload `json:"payload"`
}
