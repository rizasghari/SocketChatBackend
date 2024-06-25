package whiteboard

import (
	"github.com/redis/go-redis/v9"
	"socketChat/internal/models"
)

type SocketWhiteboardHub struct {
	// [whiteboard_id] => []*SocketClient
	Whiteboards map[uint][]*models.SocketClient
	Redis       *redis.Client
}
