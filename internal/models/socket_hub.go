package models

import (
	"github.com/redis/go-redis/v9"
	"sync"
)

type SocketHub struct {
	Clients map[uint]*SocketClient
	Mu      sync.Mutex
	Redis   *redis.Client
}
