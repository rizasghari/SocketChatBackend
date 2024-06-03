package main

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
	"sync"
	"syscall"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	clients = make(map[uint]*websocket.Conn) // Map user IDs to WebSocket connections
	rdb     *redis.Client
	ctx     = context.Background()
	mutex   sync.Mutex
)

type Message struct {
	ReceiverID uint   `json:"receiver_id"`
	Content    string `json:"content"`
}

func main() {
	// Initialize Redis client
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Initialize Gin router
	router := gin.Default()

	router.LoadHTMLGlob("./*.html")

	router.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.html", nil)
	})

	// WebSocket endpoint
	router.GET("/ws/:userID", func(ctx *gin.Context) {
		userID := ctx.Param("userID")
		log.Println("User ID:", userID)
		handleConnections(ctx.Writer, ctx.Request, userID)
	})

	// Start listening for incoming chat messages
	go handleRedisMessages()

	// Start the server
	server := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}

	go func() {
		log.Println("HTTP server started on :8000")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	waitForShutdown(server)
}

func handleConnections(w http.ResponseWriter, r *http.Request, userID string) {
	ws, err := upgrader.Upgrade(w, r, nil)
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

	mutex.Lock()
	clients[uid] = ws
	mutex.Unlock()

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading json: %v", err)
			mutex.Lock()
			delete(clients, uid)
			mutex.Unlock()
			break
		}

		// Publish the new message to Redis
		jsonMessage, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error marshalling message: %v", err)
			continue
		}

		if err := publishMessage(rdb, "chat_channel", jsonMessage); err != nil {
			log.Printf("Error publishing message: %v", err)
		}
	}
}

func handleRedisMessages() {
	ch := subscribeToChannel(rdb, "chat_channel")
	for msg := range ch {
		var message Message
		if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}

		// Send the message to the intended recipient
		sendMessageToClient(message.ReceiverID, message)
	}
}

func sendMessageToClient(receiverID uint, message Message) {
	mutex.Lock()
	defer mutex.Unlock()

	if client, ok := clients[receiverID]; ok {
		if err := client.WriteJSON(message); err != nil {
			log.Printf("Error writing json: %v", err)
			client.Close()
			delete(clients, receiverID)
		}
	}
}

func publishMessage(rdb *redis.Client, channel string, message []byte) error {
	return rdb.Publish(ctx, channel, message).Err()
}

func subscribeToChannel(rdb *redis.Client, channel string) <-chan *redis.Message {
	pubsub := rdb.Subscribe(ctx, channel)
	_, err := pubsub.Receive(ctx)
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

func waitForShutdown(server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close all WebSocket connections
	mutex.Lock()
	for uid, client := range clients {
		err := client.Close()
		if err != nil {
			return
		}
		delete(clients, uid)
	}
	mutex.Unlock()

	log.Println("Server exiting")
}
