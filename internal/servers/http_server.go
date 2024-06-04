package servers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"os"
	"os/signal"
	"socketChat/internal/models"
	"sync"
	"syscall"
)

var (
	httpServer *HttpServer
	once       sync.Once
)

type HttpServer struct {
	mu       sync.Mutex
	rdb      *redis.Client
	ctx      context.Context
	upgrader websocket.Upgrader
	clients  map[uint]*websocket.Conn
	router   *gin.Engine
}

func NewHttpServer(ctx context.Context, rdb *redis.Client) *HttpServer {
	once.Do(func() {
		httpServer = &HttpServer{
			ctx:     ctx,
			rdb:     rdb,
			clients: make(map[uint]*websocket.Conn),
		}
	})
	return httpServer
}

func (hs *HttpServer) Run() {

	hs.initializeSocketUpgrader()

	hs.initializeGin()
	hs.setupRoutes()

	// Start listening for incoming chat messages
	go hs.handleRedisMessages()

	server := hs.startServer()

	// Wait for interrupt signal to gracefully shut down the server
	hs.waitForShutdown(server)
}

func (hs *HttpServer) initializeSocketUpgrader() {
	hs.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func (hs *HttpServer) initializeGin() {
	hs.router = gin.Default()
	hs.router.LoadHTMLGlob("./*.html")
}

func (hs *HttpServer) setupRoutes() {
	hs.router.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", nil)
	})

	// WebSocket endpoint
	hs.router.GET("/ws/:userID", func(ctx *gin.Context) {
		userID := ctx.Param("userID")
		log.Println("User ID:", userID)
		hs.handleConnections(ctx.Writer, ctx.Request, userID)
	})
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

func (hs *HttpServer) handleConnections(w http.ResponseWriter, r *http.Request, userID string) {
	ws, err := hs.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer func(ws *websocket.Conn) {
		err := ws.Close()
		if err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}(ws)

	uid := parseUserID(userID)
	if uid == 0 {
		log.Printf("Invalid user ID: %v", userID)
		err := ws.Close()
		if err != nil {
			return
		}
		return
	}

	hs.mu.Lock()
	hs.clients[uid] = ws
	hs.mu.Unlock()

	for {
		var msg models.Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading json: %v", err)
			hs.mu.Lock()
			delete(hs.clients, uid)
			hs.mu.Unlock()
			break
		}

		// Publish the new message to Redis
		jsonMessage, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error marshalling message: %v", err)
			continue
		}

		if err := hs.publishMessage(hs.rdb, "chat_channel", jsonMessage); err != nil {
			log.Printf("Error publishing message: %v", err)
		}
	}
}

func (hs *HttpServer) handleRedisMessages() {
	ch := hs.subscribeToChannel(hs.rdb, "chat_channel")
	for msg := range ch {
		var message models.Message
		if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}
		// Send the message to the intended recipient
		hs.sendMessageToClient(message.ReceiverID, message)
	}
}

func (hs *HttpServer) sendMessageToClient(receiverID uint, message models.Message) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if client, ok := hs.clients[receiverID]; ok {
		if err := client.WriteJSON(message); err != nil {
			log.Printf("Error writing json: %v", err)
			err := client.Close()
			if err != nil {
				return
			}
			delete(hs.clients, receiverID)
		}
	}
}

func (hs *HttpServer) publishMessage(rdb *redis.Client, channel string, message []byte) error {
	return rdb.Publish(hs.ctx, channel, message).Err()
}

func (hs *HttpServer) subscribeToChannel(rdb *redis.Client, channel string) <-chan *redis.Message {
	pubsub := rdb.Subscribe(hs.ctx, channel)
	_, err := pubsub.Receive(hs.ctx)
	if err != nil {
		log.Fatalf("Could not subscribe to channel: %v", err)
	}
	return pubsub.Channel()
}

func parseUserID(userID string) uint {
	var uid uint
	if _, err := fmt.Sscanf(userID, "%d", &uid); err != nil {
		return 0
	}
	return uid
}

func (hs *HttpServer) waitForShutdown(httpServer *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	if err := httpServer.Shutdown(hs.ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close all WebSocket connections
	hs.mu.Lock()
	for uid, client := range hs.clients {
		err := client.Close()
		if err != nil {
			return
		}
		delete(hs.clients, uid)
	}
	hs.mu.Unlock()

	log.Println("Server exiting")
}
