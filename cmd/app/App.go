package app

import (
	"context"
	"socketChat/configs"
	"socketChat/internal/handlers"
	"socketChat/internal/repositories"
	"socketChat/internal/servers/database"
	"socketChat/internal/servers/http"
	"socketChat/internal/services"
	"sync"

	"github.com/redis/go-redis/v9"
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
	whiteboardRepo := repositories.NewWhiteboardRepository(db)
	whiteboardService := services.NewWhiteboardService(whiteboardRepo)

	minioService := services.NewMinioService(app.configs)
	fileManagerService := services.NewFileManagerService(minioService)

	restHandler := handlers.NewRestandler(
		authService,
		chatService,
		whiteboardService,
		fileManagerService,
	)

	socketChatHandler := handlers.NewSocketChatHandler(app.redis, app.ctx, chatService)
	htmlHandler := handlers.NewHtmlHandler(authService, chatService, fileManagerService)
	socketObservingHandler := handlers.NewSocketUserObservingHandler(app.redis, app.ctx, authService)
	socketWhiteboardHandler := handlers.NewSocketWhiteboardHandler(app.redis, app.ctx, whiteboardService)

	http.NewHttpServer(
		app.ctx,
		app.redis,
		restHandler,
		socketChatHandler,
		socketObservingHandler,
		socketWhiteboardHandler,
		htmlHandler,
	).Run()
}

func (app *App) initializeRedis() {
	app.redis = redis.NewClient(&redis.Options{
		Addr: "socket-chat-redis:6379",
	})
}

func (app *App) initializeConfigs() {
	app.configs = configs.GetConfig()
}
