package app

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
	app  *App
	once sync.Once
)

type App struct {
	mu       sync.Mutex
	rdb      *redis.Client
	ctx      context.Context
	upgrader websocket.Upgrader
	clients  map[uint]*websocket.Conn
	router   *gin.Engine
}

func GetApp() *App {
	once.Do(func() {
		app = &App{}
	})
	return app
}

func (app *App) LetsGo() {

	app.ctx = context.Background()
	app.clients = make(map[uint]*websocket.Conn)

	app.initializeRedis()

	app.initializeSocketUpgrader()

	app.initializeGin()
	app.setupRoutes()

	// Start listening for incoming chat messages
	go app.handleRedisMessages()

	server := app.startServer()

	// Wait for interrupt signal to gracefully shut down the server
	app.waitForShutdown(server)
}

func (app *App) initializeRedis() {
	app.rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}

func (app *App) initializeSocketUpgrader() {
	app.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func (app *App) initializeGin() {
	app.router = gin.Default()
	app.router.LoadHTMLGlob("./*.html")
}

func (app *App) setupRoutes() {
	app.router.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", nil)
	})

	// WebSocket endpoint
	app.router.GET("/ws/:userID", func(ctx *gin.Context) {
		userID := ctx.Param("userID")
		log.Println("User ID:", userID)
		app.handleConnections(ctx.Writer, ctx.Request, userID)
	})
}

func (app *App) startServer() *http.Server {
	server := &http.Server{
		Addr:    ":8000",
		Handler: app.router,
	}

	go func() {
		log.Println("HTTP server started on :8000")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	return server
}

func (app *App) handleConnections(w http.ResponseWriter, r *http.Request, userID string) {
	ws, err := app.upgrader.Upgrade(w, r, nil)
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

	app.mu.Lock()
	app.clients[uid] = ws
	app.mu.Unlock()

	for {
		var msg models.Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading json: %v", err)
			app.mu.Lock()
			delete(app.clients, uid)
			app.mu.Unlock()
			break
		}

		// Publish the new message to Redis
		jsonMessage, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error marshalling message: %v", err)
			continue
		}

		if err := app.publishMessage(app.rdb, "chat_channel", jsonMessage); err != nil {
			log.Printf("Error publishing message: %v", err)
		}
	}
}

func (app *App) handleRedisMessages() {
	ch := app.subscribeToChannel(app.rdb, "chat_channel")
	for msg := range ch {
		var message models.Message
		if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}

		// Send the message to the intended recipient
		app.sendMessageToClient(message.ReceiverID, message)
	}
}

func (app *App) sendMessageToClient(receiverID uint, message models.Message) {
	app.mu.Lock()
	defer app.mu.Unlock()

	if client, ok := app.clients[receiverID]; ok {
		if err := client.WriteJSON(message); err != nil {
			log.Printf("Error writing json: %v", err)
			err := client.Close()
			if err != nil {
				return
			}
			delete(app.clients, receiverID)
		}
	}
}

func (app *App) publishMessage(rdb *redis.Client, channel string, message []byte) error {
	return rdb.Publish(app.ctx, channel, message).Err()
}

func (app *App) subscribeToChannel(rdb *redis.Client, channel string) <-chan *redis.Message {
	pubsub := rdb.Subscribe(app.ctx, channel)
	_, err := pubsub.Receive(app.ctx)
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

func (app *App) waitForShutdown(server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	if err := server.Shutdown(app.ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close all WebSocket connections
	app.mu.Lock()
	for uid, client := range app.clients {
		err := client.Close()
		if err != nil {
			return
		}
		delete(app.clients, uid)
	}
	app.mu.Unlock()

	log.Println("Server exiting")
}
