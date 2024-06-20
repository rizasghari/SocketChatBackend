package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"socketChat/internal/enums"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	redisModels "socketChat/internal/models/redis"
	socketModels "socketChat/internal/models/socket"
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

type SocketChatHandler struct {
	mu          sync.Mutex
	ctx         context.Context
	upgrader    websocket.Upgrader
	hub         *models.SocketHub
	chatService *services.ChatService
}

func NewSocketChatHandler(redis *redis.Client, ctx context.Context, chatService *services.ChatService) *SocketChatHandler {
	return &SocketChatHandler{
		ctx:         ctx,
		chatService: chatService,
		hub: &models.SocketHub{
			Conversations: make(map[uint][]*models.SocketClient),
			Redis:         redis,
			Mu:            sync.Mutex{},
		},
	}
}

func (sch *SocketChatHandler) HandleSocketChatRoute(ctx *gin.Context) {
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

	userInfo, err := utils.VerifyToken(jwtToken)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrUnauthorized},
		})
		return
	}

	// Validate user
	// Todo: Validate user in proper way
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
	if !sch.chatService.CheckConversationExists(conversationIdUInt) {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrInvalidConversationId},
		})
		return
	}
	// Check if user is part of the conversation
	if !sch.chatService.CheckUserInConversation(userInfo.ID, conversationIdUInt) {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrInvalidConversationId},
		})
		return
	}

	sch.HandleConnections(ctx, userInfo, conversationIdUInt)
}

func (sch *SocketChatHandler) StartSocket() {
	sch.InitializeSocketUpgrader()
	go sch.HandleRedisMessages()
}

func (sch *SocketChatHandler) InitializeSocketUpgrader() {
	sch.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func (sch *SocketChatHandler) HandleConnections(ctx *gin.Context, userInfo *models.Claims, conversationId uint) {
	ws, err := sch.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
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
	sch.handleDiconnectedClient(ws, userInfo.ID, conversationId)

	// Add client to hub
	sch.handleConversationAndClinet(userInfo.ID, conversationId, ws)

	// Handle incoming messages
	sch.handleIncommingMessagesWithEvent(ws, userInfo, conversationId)
}

func (sch *SocketChatHandler) handleDiconnectedClient(ws *websocket.Conn, userId uint, conversationId uint) {
	ws.SetCloseHandler(func(code int, text string) error {
		sch.deleteDiconnectedClientFromConversation(userId, conversationId)
		return nil
	})
}

func (sch *SocketChatHandler) handleConversationAndClinet(userId uint, conversationId uint, ws *websocket.Conn) {
	sch.mu.Lock()
	// Add conversation to hub if not exists
	if _, ok := sch.hub.Conversations[conversationId]; !ok {
		sch.hub.Conversations[conversationId] = []*models.SocketClient{}
	}
	// Add client to conversation if not exists
	if isMember := slices.Contains(sch.hub.Conversations[conversationId], &models.SocketClient{Conn: ws}); !isMember {
		sch.hub.Conversations[conversationId] =
			append(sch.hub.Conversations[conversationId],
				&models.SocketClient{
					Conn:   ws,
					UserId: userId,
				},
			)
	}
	sch.mu.Unlock()

	// Log conversations for debug purposes
	sch.logConversations()
}

func (sch *SocketChatHandler) handleIncommingMessagesWithEvent(ws *websocket.Conn, userInfo *models.Claims, conversationId uint) {
	for {
		// Read message from client
		var event socketModels.SocketEvent
		err := ws.ReadJSON(&event)
		if err != nil {
			log.Printf("Error reading json: %v", err)
			sch.deleteDiconnectedClientFromConversation(userInfo.ID, conversationId)
			break
		}

		// Set event conversation id
		event.ConversationID = conversationId

		// Handle event
		switch event.Event {
		case enums.SOCKET_EVENT_SEND_MESSAGE:
			errs := sch.handleSendMessageEvent(event.Payload, enums.SOCKET_EVENT_SEND_MESSAGE, userInfo, conversationId)
			if len(errs) > 0 {
				log.Printf("handleIncommingMessagesWithEvent - Error while handling send message event: %v", errs)
			}
		case enums.SOCKET_EVENT_SEEN_MESSAGE:
			errs := sch.handleSeenMessageEvent(event.Payload, enums.SOCKET_EVENT_SEEN_MESSAGE, conversationId, userInfo.ID)
			if len(errs) > 0 {
				log.Printf("handleIncommingMessagesWithEvent - Error while handling seen message event: %v", errs)
			}
		case enums.SOCKET_EVENT_IS_TYPING:
			errs := sch.handleIsTypingEvent(event.Payload, enums.SOCKET_EVENT_IS_TYPING, conversationId)
			if len(errs) > 0 {
				log.Printf("handleIncommingMessagesWithEvent - Error while handling is typing event: %v", errs)
			}
		default:
			log.Printf("Unknown event: %v", event)
		}
	}
}

func (sch *SocketChatHandler) handleIsTypingEvent(payload json.RawMessage, event string, conversationId uint) []error {
	var errors []error
	var isTypingPayload socketModels.IsTypingPayload
	err := json.Unmarshal(payload, &isTypingPayload)
	if err != nil {
		errors = append(errors, errs.ErrInvalidRequest)
		return errors
	}

	log.Println("isTypingPayload: ", isTypingPayload)

	// Publish the new message to Redis
	redisEvent := redisModels.RedisPublishedMessage{
		Event:          event,
		ConversationID: conversationId,
		Payload:        isTypingPayload,
	}

	jsonEvent, err := json.Marshal(redisEvent)
	if err != nil {
		errors = append(errors, err)
		return errors
	}
	log.Println("jsonEvent: ", string(jsonEvent))
	if err := sch.PublishMessage(sch.hub.Redis, redisModels.REDIS_CHANNEL_CHAT, jsonEvent); err != nil {
		errors = append(errors, err)
		return errors
	}
	return nil
}

func (sch *SocketChatHandler) handleSendMessageEvent(payload json.RawMessage, event string, userInfo *models.Claims, conversationId uint) []error {
	var errors []error
	var messageRequest models.MessageRequest
	err := json.Unmarshal(payload, &messageRequest)
	if err != nil {
		errors = append(errors, errs.ErrInvalidRequest)
		sch.deleteDiconnectedClientFromConversation(userInfo.ID, conversationId)
		return errors
	}

	// Save message in DB
	message := &models.Message{
		ConversationID: conversationId,
		Content:        messageRequest.Content,
		SenderID:       userInfo.ID,
	}
	savedMessage, saveMsgErrs := sch.chatService.SaveMessage(message)
	if len(saveMsgErrs) > 0 {
		errors = append(errors, saveMsgErrs...)
		return errors
	}

	// Publish the new message to Redis
	redisEvent := redisModels.RedisPublishedMessage{
		Event:          event,
		ConversationID: conversationId,
		Payload:        savedMessage,
	}

	jsonEvent, err := json.Marshal(redisEvent)
	if err != nil {
		errors = append(errors, err)
		return errors
	}
	if err := sch.PublishMessage(sch.hub.Redis, redisModels.REDIS_CHANNEL_CHAT, jsonEvent); err != nil {
		errors = append(errors, err)
		return errors
	}
	return nil
}

func (sch *SocketChatHandler) handleSeenMessageEvent(payload json.RawMessage, event string, conversationId, seenerId uint) []error {
	var errors []error
	var seenData socketModels.SeenMessagePayload
	err := json.Unmarshal(payload, &seenData)
	if err != nil {
		errors = append(errors, err)
		return errors
	}

	log.Println("seenData: ", seenData)

	// Mark message as seen in DB
	errs := sch.chatService.SeenMessage(seenData.MessageIds, seenerId)
	if len(errs) > 0 {
		errors = append(errors, errs...)
		return errors
	}
	// Publish the new message to Redis
	redisEvent := redisModels.RedisPublishedMessage{
		Event:          event,
		ConversationID: conversationId,
		Payload:        seenData,
	}

	// Publish the new message to Redis
	jsonEvent, err := json.Marshal(redisEvent)
	if err != nil {
		errors = append(errors, err)
		return errors
	}
	if err := sch.PublishMessage(sch.hub.Redis, redisModels.REDIS_CHANNEL_CHAT, jsonEvent); err != nil {
		errors = append(errors, err)
		return errors
	}
	return nil
}

func (sch *SocketChatHandler) deleteDiconnectedClientFromConversation(userId uint, conversationId uint) {
	sch.mu.Lock()
	defer sch.mu.Unlock()
	// Remove disconnected client from conversation
	for i, client := range sch.hub.Conversations[conversationId] {
		if client.UserId == userId {
			sch.hub.Conversations[conversationId] = append(sch.hub.Conversations[conversationId][:i], sch.hub.Conversations[conversationId][i+1:]...)
			break
		}
	}
	// Check if the conversation is empty and remove it from the map
	if len(sch.hub.Conversations[conversationId]) == 0 {
		delete(sch.hub.Conversations, conversationId)
	}
	// Log conversations for debug purposes
	sch.logConversations()
}

func (sch *SocketChatHandler) logConversations() {
	for conversationId, clients := range sch.hub.Conversations {
		log.Printf("Conversation ID: %v", conversationId)
		for _, client := range clients {
			log.Printf("Client ID: %v", client.UserId)
		}
	}
}

func (sch *SocketChatHandler) HandleRedisMessages() {
	ch := sch.SubscribeToChannel(sch.hub.Redis, redisModels.REDIS_CHANNEL_CHAT)
	for msg := range ch {
		var redisMessage redisModels.RedisPublishedMessage
		if err := json.Unmarshal([]byte(msg.Payload), &redisMessage); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}
		sch.SendMessageToClient(redisMessage)
	}
}

func (sch *SocketChatHandler) SendMessageToClient(redisMessage redisModels.RedisPublishedMessage) {
	sch.mu.Lock()
	defer sch.mu.Unlock()
	if conversation, ok := sch.hub.Conversations[redisMessage.ConversationID]; ok {
		for _, client := range conversation {
			if err := client.Conn.WriteJSON(redisMessage); err != nil {
				log.Printf("Error writing json: %v", err)
				err := client.Conn.Close()
				if err != nil {
					return
				}
				sch.deleteDiconnectedClientFromConversation(client.UserId, redisMessage.ConversationID)
			}
		}
	}
}

func (sch *SocketChatHandler) PublishMessage(redis *redis.Client, channel string, message []byte) error {
	return redis.Publish(sch.ctx, channel, message).Err()
}

func (sch *SocketChatHandler) SubscribeToChannel(redis *redis.Client, channel string) <-chan *redis.Message {
	pubsub := redis.Subscribe(sch.ctx, channel)
	_, err := pubsub.Receive(sch.ctx)
	if err != nil {
		log.Fatalf("Could not subscribe to channel: %v", err)
	}
	return pubsub.Channel()
}

func (sch *SocketChatHandler) WaitForShutdown(httpServer *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	if err := httpServer.Shutdown(sch.ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close all WebSocket connections
	sch.mu.Lock()
	for conversationId, clients := range sch.hub.Conversations {
		for _, client := range clients {
			err := client.Conn.Close()
			if err != nil {
				return
			}
		}
		delete(sch.hub.Conversations, conversationId)
	}
	sch.mu.Unlock()

	log.Println("Server exiting")
}
