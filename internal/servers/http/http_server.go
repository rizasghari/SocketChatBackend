package http

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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
	handler *handlers.Handler
	socket  *handlers.Socket
	redis   *redis.Client
	ctx     context.Context
}

func NewHttpServer(ctx context.Context, redis *redis.Client, handler *handlers.Handler) *HttpServer {
	once.Do(func() {
		httpServer = &HttpServer{
			handler: handler,
			redis:   redis,
			ctx:     ctx,
			socket: handlers.NewSocket(redis, ctx),
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
	hs.router.GET("/", hs.handler.Index)
	hs.router.POST("/login", hs.handler.Login)
	hs.router.POST("/register", hs.handler.Register)
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
