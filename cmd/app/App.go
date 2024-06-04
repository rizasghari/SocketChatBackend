package app

import (
	"context"
	"github.com/redis/go-redis/v9"
	"socketChat/internal/servers"
	"sync"
)

var (
	app  *App
	once sync.Once
)

type App struct {
	rdb *redis.Client
	ctx context.Context
}

func GetApp() *App {
	once.Do(func() {
		app = &App{}
	})
	return app
}

func (app *App) LetsGo() {
	app.ctx = context.Background()
	app.initializeRedis()
	servers.NewHttpServer(app.ctx, app.rdb).Run()
}

func (app *App) initializeRedis() {
	app.rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}
