package http

import (
	"context"
	"errors"
	"log"
	"net/http"
	"socketChat/internal/handlers"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var (
	httpServer *HttpServer
	once       sync.Once
)

type HttpServer struct {
	router                     *gin.Engine
	restHandler                *handlers.RestHandler
	htmlHandler                *handlers.HtmlHandler
	socketChatHandler          *handlers.SocketChatHandler
	socketUserObservingHandler *handlers.SocketUserObservingHandler
	socketWhiteboardHandler    *handlers.SocketWhiteboardHandler
	redis                      *redis.Client
	ctx                        context.Context
}

func NewHttpServer(
	ctx context.Context,
	redis *redis.Client,
	restHandler *handlers.RestHandler,
	socketChatHandler *handlers.SocketChatHandler,
	socketUserObservingHandler *handlers.SocketUserObservingHandler,
	socketWhiteboardHandler *handlers.SocketWhiteboardHandler,
	htmlHandler *handlers.HtmlHandler,
) *HttpServer {
	once.Do(func() {
		httpServer = &HttpServer{
			restHandler:                restHandler,
			redis:                      redis,
			ctx:                        ctx,
			socketChatHandler:          socketChatHandler,
			socketUserObservingHandler: socketUserObservingHandler,
			socketWhiteboardHandler:    socketWhiteboardHandler,
			htmlHandler:                htmlHandler,
		}
	})
	return httpServer
}

func (hs *HttpServer) Run() {
	hs.initializeGin()
	hs.setupWebSocketRoutes()
	hs.setupRestfulRoutes()
	hs.socketChatHandler.StartSocket()
	server := hs.startServer()
	
	hs.socketChatHandler.WaitForShutdown(server)
	hs.socketWhiteboardHandler.WaitForShutdown(server)
}

func (hs *HttpServer) initializeGin() {
	hs.router = gin.Default()

	hs.router.Static("/web/static", "./web/static")

	ginHtmlRenderer := hs.router.HTMLRender
	hs.router.HTMLRender = &handlers.HTMLTemplRenderer{FallbackHtmlRenderer: ginHtmlRenderer}

	// Disable trusted proxy warning.
	err := hs.router.SetTrustedProxies(nil)
	if err != nil {
		return
	}
}

func (hs *HttpServer) setupRestfulRoutes() {
	// Handle no route found
	hs.router.NoRoute(hs.htmlHandler.NotFound)

	// Apply the CORS middleware to the router
	hs.router.Use(handlers.CORSMiddleware())

	web := hs.router.Group("/")
	{
		web.GET("/", hs.htmlHandler.Index)
		web.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	v1 := hs.router.Group("/api/v1")
	{
		v1.POST("/login", hs.restHandler.Login)
		v1.POST("/register", hs.restHandler.Register)
	}

	authenticated := v1.Group("/")
	authenticated.Use(handlers.MustAuthenticateMiddleware())
	{
		authenticated.GET("/users", hs.restHandler.GetAllUsersWithPagination)
		authenticated.GET("/users/:id", hs.restHandler.GetSingleUser)
		authenticated.POST("/users/upload-profile-photo", hs.restHandler.UploadUserProfilePhoto)
		authenticated.PUT("/users", hs.restHandler.UpdateUser)
		authenticated.GET("/users/discover", hs.restHandler.DiscoverUsers)
		authenticated.GET("/profile", hs.restHandler.GetUserProfile)
		authenticated.GET("/users/sent-message/:concurrent/:mutex", hs.restHandler.GetUsersWhoHaveSentMessage)

		authenticated.POST("/conversations", hs.restHandler.CreateConversation)
		authenticated.GET("/conversations/user/:id", hs.restHandler.GetUserConversations)
		authenticated.GET("/conversations/my", hs.restHandler.GetUserConversationsByToken)
		authenticated.GET("/conversations/unread/:id", hs.restHandler.GetConversationUnReadMessagesForUser)

		authenticated.POST("/messages", hs.restHandler.SaveMessage)
		authenticated.GET("/messages/conversation/:id", hs.restHandler.GetMessagesByConversationID)

		authenticated.POST("/whiteboards", hs.restHandler.CreateWhiteboard)
	}
}

func (hs *HttpServer) setupWebSocketRoutes() {
	hs.router.GET("/ws/chat", hs.socketChatHandler.HandleSocketChatRoute)
	hs.router.GET("/ws/observe", hs.socketUserObservingHandler.HandleSocketUserObservingRoute)
	hs.router.GET("/ws/whiteboard", hs.socketWhiteboardHandler.HandleSocketWhiteboardRoute)
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
