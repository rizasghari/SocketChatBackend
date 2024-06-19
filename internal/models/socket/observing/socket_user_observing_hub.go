package observing

import (
	"github.com/redis/go-redis/v9"
	"socketChat/internal/models"
	"sync"
)

type SocketUserObservingHub struct {
	// [user_id] => []*SocketClient
	Notifiers map[uint][]*models.SocketClient
	Mu        sync.Mutex
	Redis     *redis.Client
}
