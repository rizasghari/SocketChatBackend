package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	"socketChat/internal/msgs"
	"socketChat/internal/utils"
	"sync"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type Socket struct {
	mu       sync.Mutex
	ctx      context.Context
	upgrader websocket.Upgrader
	hub      *models.SocketHub
}

func NewSocket(redis *redis.Client, ctx context.Context) *Socket {
	return &Socket{
		ctx: ctx,
		hub: &models.SocketHub{
			Clients: make(map[uint]*models.SocketClient),
			Redis:   redis,
			Mu:      sync.Mutex{},
		},
	}
}

func (s *Socket) HandleSocketRoute(ctx *gin.Context) {
	jwtToken := ctx.Request.Header.Get("Authorization")

	// Authenticate
	if jwtToken == "" {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrUnauthorized},
		})
		return
	}

	claims, err := utils.VerifyToken(jwtToken, utils.GetJwtKey())
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrUnauthorized},
		})
		return
	}

	s.HandleConnections(ctx, claims)

}

func (s *Socket) StartSocket() {
	s.InitializeSocketUpgrader()
	go s.HandleRedisMessages()
}

func (s *Socket) InitializeSocketUpgrader() {
	s.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func (s *Socket) HandleConnections(ctx *gin.Context, userInfo *models.Claims) {
	ws, err := s.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
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

	userId := userInfo.ID
	if userId == 0 {
		log.Printf("Invalid user ID: %v", userId)
		err := ws.Close()
		if err != nil {
			return
		}
		return
	}

	s.mu.Lock()
	s.hub.Clients[userId] = &models.SocketClient{
		Conn:           ws,
		ConversationID: userId,
	}
	s.mu.Unlock()

	for id, client := range s.hub.Clients {
		log.Println("Client connected:", id, client.Conn.RemoteAddr())
	}

	for {
		var msg models.TempSocketMessage
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading json: %v", err)
			s.mu.Lock()
			delete(s.hub.Clients, userId)
			s.mu.Unlock()
			break
		}

		// Publish the new message to Redis
		jsonMessage, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error marshalling message: %v", err)
			continue
		}
		if err := s.PublishMessage(s.hub.Redis, "chat_channel", jsonMessage); err != nil {
			log.Printf("Error publishing message: %v", err)
		}
	}
}

func (s *Socket) HandleRedisMessages() {
	ch := s.SubscribeToChannel(s.hub.Redis, "chat_channel")
	for msg := range ch {
		var message models.TempSocketMessage
		if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}
		// Send the message to the intended recipient
		s.SendMessageToClient(message.ReceiverID, message)
	}
}

func (s *Socket) SendMessageToClient(receiverID uint, message models.TempSocketMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, ok := s.hub.Clients[receiverID]; ok {
		if err := client.Conn.WriteJSON(message); err != nil {
			log.Printf("Error writing json: %v", err)
			err := client.Conn.Close()
			if err != nil {
				return
			}
			delete(s.hub.Clients, receiverID)
		}
	}
}

func (s *Socket) PublishMessage(redis *redis.Client, channel string, message []byte) error {
	return redis.Publish(s.ctx, channel, message).Err()
}

func (hs *Socket) SubscribeToChannel(redis *redis.Client, channel string) <-chan *redis.Message {
	pubsub := redis.Subscribe(hs.ctx, channel)
	_, err := pubsub.Receive(hs.ctx)
	if err != nil {
		log.Fatalf("Could not subscribe to channel: %v", err)
	}
	return pubsub.Channel()
}

func (s *Socket) WaitForShutdown(httpServer *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	if err := httpServer.Shutdown(s.ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close all WebSocket connections
	s.mu.Lock()
	for uid, client := range s.hub.Clients {
		err := client.Conn.Close()
		if err != nil {
			return
		}
		delete(s.hub.Clients, uid)
	}
	s.mu.Unlock()

	log.Println("Server exiting")
}
