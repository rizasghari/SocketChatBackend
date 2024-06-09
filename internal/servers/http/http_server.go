package http

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/swaggo/files"      
	"github.com/swaggo/gin-swagger"
	"log"
	"net/http"
	"socketChat/internal/handlers"
	"sync"
)

var (
	httpServer *HttpServer
	once       sync.Once
)

type HttpServer struct {
	router  *gin.Engine
	handler *handlers.RestHandler
	socket  *handlers.SocketHandler
	redis   *redis.Client
	ctx     context.Context
}

func NewHttpServer(ctx context.Context, redis *redis.Client, handler *handlers.RestHandler, socket *handlers.SocketHandler) *HttpServer {
	once.Do(func() {
		httpServer = &HttpServer{
			handler: handler,
			redis:   redis,
			ctx:     ctx,
			socket:  socket,
		}
	})
	return httpServer
}

func (hs *HttpServer) Run() {
	hs.initializeGin()
	hs.setupWebSocketRoutes()
	hs.setupRestfulRoutes()
	hs.socket.StartSocket()
	server := hs.startServer()
	// Wait for interrupt signal to gracefully shut down the server
	hs.socket.WaitForShutdown(server)
}

func (hs *HttpServer) initializeGin() {
	hs.router = gin.Default()
	hs.router.LoadHTMLGlob("./*.html")
}


func (hs *HttpServer) setupRestfulRoutes() {
	web := hs.router.Group("/")
	{
		web.GET("/", hs.handler.Index)
		web.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	v1 := hs.router.Group("/api/v1")
	{
		v1.POST("/login", hs.handler.Login)
		v1.POST("/register", hs.handler.Register)
	}

	authenticated := v1.Group("/")
	authenticated.Use(hs.handler.MustAuthenticateMiddleware())
	{
		authenticated.GET("/users", hs.handler.GetAllUsersWithPagination)
		authenticated.GET("/users/:id", hs.handler.GetSingleUser)
		authenticated.POST("/users/upload-profile-photo", hs.handler.UploadUserProfilePhoto)
		authenticated.PUT("/users", hs.handler.UpdateUser)

		authenticated.POST("/conversations", hs.handler.CreateConversation)
		authenticated.GET("/conversations/user/:id", hs.handler.GetUserConversations)
		authenticated.GET("/conversations/my", hs.handler.GetUserConversationsByToken)

		authenticated.POST("/messages", hs.handler.SaveMessage)
		authenticated.GET("/messages/conversation/:id", hs.handler.GetMessagesByConversationID)
	}
}

func (hs *HttpServer) setupWebSocketRoutes() {
	hs.router.GET("/ws", hs.socket.HandleSocketRoute)
}

func (hs *HttpServer) startServer() *http.Server {
	server := &http.Server{
		Addr:    ":8000",
		Handler: hs.router,
	}

	go func() {
		log.Println("HTTP server started on :8000")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	return server
}
