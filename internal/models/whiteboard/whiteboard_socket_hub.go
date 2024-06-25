package whiteboard

import (
	"socketChat/internal/models"
)

type SocketWhiteboardHub struct {
	// [whiteboard_id] => []*SocketClient
	Whiteboards map[uint][]*models.SocketClient
}
