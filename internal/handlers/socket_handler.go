package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	"socketChat/internal/msgs"
	"socketChat/internal/services"
	"socketChat/internal/utils"
	"strconv"
	"sync"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type SocketHandler struct {
	mu          sync.Mutex
	ctx         context.Context
	upgrader    websocket.Upgrader
	hub         *models.SocketHub
	chatService *services.ChatService
}

func NewSocketHandler(redis *redis.Client, ctx context.Context, chatService *services.ChatService) *SocketHandler {
	return &SocketHandler{
		ctx:         ctx,
		chatService: chatService,
		hub: &models.SocketHub{
			Conversations: make(map[uint][]*models.SocketClient),
			Redis:         redis,
			Mu:            sync.Mutex{},
		},
	}
}

func (sh *SocketHandler) HandleSocketRoute(ctx *gin.Context) {
	// Authenticate user
	jwtToken := ctx.Request.Header.Get("Authorization")
	if jwtToken == "" {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrUnauthorized},
		})
		return
	}

	userInfo, err := utils.VerifyToken(jwtToken, utils.GetJwtKey())
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrUnauthorized},
		})
		return
	}

	// Validate user
	if userInfo.ID == 0 {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrUnauthorized},
		})
		return
	}

	// Get conversation ID and validate if it exists
	conversationId := ctx.Query("conversationId")
	if conversationId == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrInvalidConversationId},
		})
		return
	}
	conversationIdInt, err := strconv.Atoi(conversationId)
	if err != nil || conversationIdInt == 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrInvalidConversationId},
		})
		return
	}
	conversationIdUInt := uint(conversationIdInt)
	if !sh.chatService.CheckConversationExists(conversationIdUInt) {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrInvalidConversationId},
		})
		return
	}
	// Check if user is part of the conversation
	if !sh.chatService.CheckUserInConversation(userInfo.ID, conversationIdUInt) {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrInvalidConversationId},
		})
		return
	}

	sh.HandleConnections(ctx, userInfo, conversationIdUInt)
}

func (sh *SocketHandler) StartSocket() {
	sh.InitializeSocketUpgrader()
	go sh.HandleRedisMessages()
}

func (sh *SocketHandler) InitializeSocketUpgrader() {
	sh.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func (sh *SocketHandler) HandleConnections(ctx *gin.Context, userInfo *models.Claims, conversationId uint) {
	ws, err := sh.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
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

	// Handle disconnection
	sh.handleDiconnectedClient(ws, userInfo.ID, conversationId)

	// Add client to hub
	sh.addConversationToHub(userInfo.ID, conversationId, ws)

	// Handle incoming messages
	sh.handleIncommingMessages(ws, userInfo, conversationId)
}

func (sh *SocketHandler) handleDiconnectedClient(ws *websocket.Conn, userId uint, conversationId uint) {
	ws.SetCloseHandler(func(code int, text string) error {
		sh.mu.Lock()
		sh.deleteDiconnectedClientFromConversation(userId, conversationId)
		sh.mu.Unlock()
		return nil
	})
}

func (sh *SocketHandler) addConversationToHub(userId uint, conversationId uint, ws *websocket.Conn) {
	sh.mu.Lock()
	// Add conversation to hub if not exists
	if _, ok := sh.hub.Conversations[conversationId]; !ok {
		sh.hub.Conversations[conversationId] = []*models.SocketClient{}
	}
	// Add client to conversation if not exists
	if isMember := slices.Contains(sh.hub.Conversations[conversationId], &models.SocketClient{Conn: ws}); !isMember {
		sh.hub.Conversations[conversationId] =
			append(sh.hub.Conversations[conversationId],
				&models.SocketClient{
					Conn:   ws,
					UserId: userId,
				},
			)
	}
	sh.mu.Unlock()

	// Log conversations for debug purposes
	sh.logConversations()
}

func (sh *SocketHandler) handleIncommingMessages(ws *websocket.Conn, userInfo *models.Claims, conversationId uint) {
	for {
		// Read message from client
		var messageRequest models.MessageRequest
		err := ws.ReadJSON(&messageRequest)
		if err != nil {
			log.Printf("Error reading json: %v", err)
			sh.mu.Lock()
			sh.deleteDiconnectedClientFromConversation(userInfo.ID, conversationId)
			sh.mu.Unlock()
			break
		}

		// Save message in DB
		message := &models.Message{
			ConversationID: messageRequest.ConversationID,
			Content:        messageRequest.Content,
			SenderID:       userInfo.ID,
		}
		savedMessage, saveMsgErrs := sh.chatService.SaveMessage(message)
		if len(saveMsgErrs) > 0 {
			log.Printf("Error saving message: %v", saveMsgErrs)
			break
		}

		// Publish the new message to Redis
		jsonMessage, err := json.Marshal(savedMessage)
		if err != nil {
			log.Printf("Error marshalling message: %v", err)
			continue
		}
		if err := sh.PublishMessage(sh.hub.Redis, "chat_channel", jsonMessage); err != nil {
			log.Printf("Error publishing message: %v", err)
		}
	}
}

func (sh *SocketHandler) deleteDiconnectedClientFromConversation(userId uint, conversationId uint) {
	// Remove disconnected client from conversation
	for i, client := range sh.hub.Conversations[conversationId] {
		if client.UserId == userId {
			sh.hub.Conversations[conversationId] = append(sh.hub.Conversations[conversationId][:i], sh.hub.Conversations[conversationId][i+1:]...)
			break
		}
	}
	// Check if the conversation is empty and remove it from the map
	if len(sh.hub.Conversations[conversationId]) == 0 {
		delete(sh.hub.Conversations, conversationId)
	}
	// Log conversations for debug purposes
	sh.logConversations()
}

func (sh *SocketHandler) logConversations() {
	for conversationId, clients := range sh.hub.Conversations {
		log.Printf("Conversation ID: %v", conversationId)
		for _, client := range clients {
			log.Printf("Client ID: %v, Connection: %v", client.UserId, client.Conn)
		}
	}
}

func (sh *SocketHandler) HandleRedisMessages() {
	ch := sh.SubscribeToChannel(sh.hub.Redis, "chat_channel")
	for msg := range ch {
		var message models.Message
		if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}
		// Send the message to the intended recipient
		sh.SendMessageToClient(message)
	}
}

func (sh *SocketHandler) SendMessageToClient(message models.Message) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if conversation, ok := sh.hub.Conversations[message.ConversationID]; ok {
		for _, client := range conversation {
			if err := client.Conn.WriteJSON(message); err != nil {
				log.Printf("Error writing json: %v", err)
				err := client.Conn.Close()
				if err != nil {
					return
				}
				sh.deleteDiconnectedClientFromConversation(client.UserId, message.ConversationID)
			}
		}
	}
}

func (sh *SocketHandler) PublishMessage(redis *redis.Client, channel string, message []byte) error {
	return redis.Publish(sh.ctx, channel, message).Err()
}

func (sh *SocketHandler) SubscribeToChannel(redis *redis.Client, channel string) <-chan *redis.Message {
	pubsub := redis.Subscribe(sh.ctx, channel)
	_, err := pubsub.Receive(sh.ctx)
	if err != nil {
		log.Fatalf("Could not subscribe to channel: %v", err)
	}
	return pubsub.Channel()
}

func (sh *SocketHandler) WaitForShutdown(httpServer *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	if err := httpServer.Shutdown(sh.ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close all WebSocket connections
	sh.mu.Lock()
	for conversationId, clients := range sh.hub.Conversations {
		for _, client := range clients {
			err := client.Conn.Close()
			if err != nil {
				return
			}
		}
		delete(sh.hub.Conversations, conversationId)
	}
	sh.mu.Unlock()

	log.Println("Server exiting")
}
