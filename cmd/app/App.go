package app

import (
	"context"
	"github.com/redis/go-redis/v9"
	"socketChat/configs"
	"socketChat/internal/handlers"
	"socketChat/internal/repositories"
	"socketChat/internal/servers/database"
	"socketChat/internal/servers/http"
	"socketChat/internal/services"
	"sync"
)

var (
	app  *App
	once sync.Once
)

type App struct {
	redis   *redis.Client
	ctx     context.Context
	configs *configs.Config
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
	app.initializeConfigs()

	db := database.GetDB(app.configs)
	authRepo := repositories.NewAuthenticationRepository(db)
	authService := services.NewAuthenticationService(authRepo, app.configs)
	chatRepo := repositories.NewChatRepository(db)
	chatService := services.NewChatService(chatRepo)
	handler := handlers.NewHandler(authService, chatService)

	http.NewHttpServer(app.ctx, app.redis, handler).Run()
}

func (app *App) initializeRedis() {
	app.redis = redis.NewClient(&redis.Options{
		Addr: "socket-chat-redis:6379",
	})
}

func (app *App) initializeConfigs() {
	app.configs = configs.GetConfig()
}
