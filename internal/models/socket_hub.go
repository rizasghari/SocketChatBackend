package models

import (
	"github.com/redis/go-redis/v9"
	"sync"
)

type SocketHub struct {
	// [conversation_id] => []*SocketClient
	Conversations map[uint][]*SocketClient
	Mu            sync.Mutex
	Redis         *redis.Client
}
