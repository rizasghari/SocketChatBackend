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
	"socketChat/internal/models/whiteboard"
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

type SocketWhiteboardHandler struct {
	mu                sync.Mutex
	ctx               context.Context
	upgrader          websocket.Upgrader
	hub               *whiteboard.SocketWhiteboardHub
	whiteboardService *services.WhiteboardService
}

func NewSocketWhiteboardHandler(redis *redis.Client, ctx context.Context, whiteboardService *services.WhiteboardService) *SocketWhiteboardHandler {
	return &SocketWhiteboardHandler{
		ctx:               ctx,
		whiteboardService: whiteboardService,
		mu:                sync.Mutex{},
		hub: &whiteboard.SocketWhiteboardHub{
			Whiteboards: make(map[uint][]*models.SocketClient),
			Redis:       redis,
		},
	}
}

func (swh *SocketWhiteboardHandler) HandleSocketWhiteboardRoute(ctx *gin.Context) {
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
	whiteboardIdStr := ctx.Query("whiteboardId")
	if whiteboardIdStr == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrInvalidParams},
		})
		return
	}
	whiteboardIdInt, err := strconv.Atoi(whiteboardIdStr)
	if err != nil || whiteboardIdInt == 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrInvalidwhiteboardId},
		})
		return
	}
	whiteboardId := uint(whiteboardIdInt)

	// Todo: Check if whiteboard exists

	// Todo: check if user is whiteboard member

	swh.HandleConnections(ctx, userInfo, whiteboardId)
}

func (swh *SocketWhiteboardHandler) StartSocket() {
	swh.InitializeSocketUpgrader()
	go swh.HandleRedisMessages()
}

func (swh *SocketWhiteboardHandler) InitializeSocketUpgrader() {
	swh.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func (swh *SocketWhiteboardHandler) HandleConnections(ctx *gin.Context, userInfo *models.Claims, whiteboardId uint) {
	ws, err := swh.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
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
	swh.handleDiconnectedClient(ws, userInfo.ID, whiteboardId)

	// Add client to hub
	swh.handleConversationAndClinet(userInfo.ID, whiteboardId, ws)

	// Handle incoming messages
	swh.handleIncommingMessagesWithEvent(ws, userInfo, whiteboardId)
}

func (swh *SocketWhiteboardHandler) handleDiconnectedClient(ws *websocket.Conn, userId uint, whiteboardId uint) {
	ws.SetCloseHandler(func(code int, text string) error {
		swh.deleteDiconnectedClientFromWhiteboard(userId, whiteboardId)
		return nil
	})
}

func (swh *SocketWhiteboardHandler) handleConversationAndClinet(userId uint, whiteboardId uint, ws *websocket.Conn) {
	swh.mu.Lock()
	// Add conversation to hub if not exists
	if _, ok := swh.hub.Whiteboards[whiteboardId]; !ok {
		swh.hub.Whiteboards[whiteboardId] = []*models.SocketClient{}
	}
	// Add client to conversation if not exists
	if isMember := slices.Contains(swh.hub.Whiteboards[whiteboardId], &models.SocketClient{Conn: ws}); !isMember {
		swh.hub.Whiteboards[whiteboardId] =
			append(swh.hub.Whiteboards[whiteboardId],
				&models.SocketClient{
					Conn:   ws,
					UserId: userId,
				},
			)
	}
	swh.mu.Unlock()

	// Log conversations for debug purposes
	swh.logConversations()
}

func (swh *SocketWhiteboardHandler) handleIncommingMessagesWithEvent(ws *websocket.Conn, userInfo *models.Claims, whiteboardId uint) {
	for {
		// Read message from client
		var event whiteboard.WhiteboardSocketEvent
		err := ws.ReadJSON(&event)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
				swh.deleteDiconnectedClientFromWhiteboard(userInfo.ID, whiteboardId)
				break
			}
			log.Printf("Error reading json: %v", err)
		}

		// Set event conversation id
		event.Payload.WhiteboardId = whiteboardId

		// Handle event
		switch event.Event {
		case enums.SOCKET_EVENT_UPDATE_WHITEBOARD:
			errs := swh.handleUpdateWhiteboardEvent(event.Payload, enums.SOCKET_EVENT_UPDATE_WHITEBOARD, userInfo, whiteboardId)
			if len(errs) > 0 {
				log.Printf("handleIncommingMessagesWithEvent - Error while handling send message event: %v", errs)
			}
		default:
			log.Printf("Unknown event: %v", event)
		}
	}
}

func (swh *SocketWhiteboardHandler) handleUpdateWhiteboardEvent(payload whiteboard.WhiteboardSocketPayload,
	event string, userInfo *models.Claims, whiteboardId uint) []error {

	var errors []error
	log.Println("handleUpdateWhiteboardEvent payload: ", payload)

	// Publish the new message to Redis
	redisEvent := whiteboard.WhiteboardSocketEvent{
		Event:   event,
		Payload: payload,
	}

	jsonEvent, err := json.Marshal(redisEvent)
	if err != nil {
		errors = append(errors, err)
		return errors
	}
	log.Println("jsonEvent: ", string(jsonEvent))
	if err := swh.PublishMessage(swh.hub.Redis, redisModels.REDIS_CHANNEL_WHITEBOARD, jsonEvent); err != nil {
		errors = append(errors, err)
		return errors
	}
	return nil
}

func (swh *SocketWhiteboardHandler) deleteDiconnectedClientFromWhiteboard(userId uint, whiteboardId uint) {
	swh.mu.Lock()
	defer swh.mu.Unlock()
	// Remove disconnected client from conversation
	for i, client := range swh.hub.Whiteboards[whiteboardId] {
		if client.UserId == userId {
			swh.hub.Whiteboards[whiteboardId] = append(swh.hub.Whiteboards[whiteboardId][:i], swh.hub.Whiteboards[whiteboardId][i+1:]...)
			break
		}
	}
	// Check if the conversation is empty and remove it from the map
	if len(swh.hub.Whiteboards[whiteboardId]) == 0 {
		delete(swh.hub.Whiteboards, whiteboardId)
	}
	// Log conversations for debug purposes
	swh.logConversations()
}

func (swh *SocketWhiteboardHandler) logConversations() {
	for whiteboardId, clients := range swh.hub.Whiteboards {
		log.Printf("Conversation ID: %v", whiteboardId)
		for _, client := range clients {
			log.Printf("Client ID: %v", client.UserId)
		}
	}
}

func (swh *SocketWhiteboardHandler) HandleRedisMessages() {
	ch := swh.SubscribeToChannel(swh.hub.Redis, redisModels.REDIS_CHANNEL_CHAT)
	for msg := range ch {
		var redisMessage whiteboard.WhiteboardSocketPayload
		if err := json.Unmarshal([]byte(msg.Payload), &redisMessage); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}
		swh.SendMessageToClient(redisMessage)
	}
}

func (swh *SocketWhiteboardHandler) SendMessageToClient(redisMessage whiteboard.WhiteboardSocketPayload) {
	swh.mu.Lock()
	defer swh.mu.Unlock()
	if whiteboard, ok := swh.hub.Whiteboards[redisMessage.WhiteboardId]; ok {
		for _, client := range whiteboard {
			if err := client.Conn.WriteJSON(redisMessage); err != nil {
				log.Printf("Error writing json: %v", err)
				err := client.Conn.Close()
				if err != nil {
					return
				}
				swh.deleteDiconnectedClientFromWhiteboard(client.UserId, redisMessage.WhiteboardId)
			}
		}
	}
}

func (swh *SocketWhiteboardHandler) PublishMessage(redis *redis.Client, channel string, message []byte) error {
	return redis.Publish(swh.ctx, channel, message).Err()
}

func (swh *SocketWhiteboardHandler) SubscribeToChannel(redis *redis.Client, channel string) <-chan *redis.Message {
	pubsub := redis.Subscribe(swh.ctx, channel)
	_, err := pubsub.Receive(swh.ctx)
	if err != nil {
		log.Fatalf("Could not subscribe to channel: %v", err)
	}
	return pubsub.Channel()
}

func (swh *SocketWhiteboardHandler) WaitForShutdown(httpServer *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	if err := httpServer.Shutdown(swh.ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close all WebSocket connections
	swh.mu.Lock()
	for whiteboardId, clients := range swh.hub.Whiteboards {
		for _, client := range clients {
			err := client.Conn.Close()
			if err != nil {
				return
			}
		}
		delete(swh.hub.Whiteboards, whiteboardId)
	}
	swh.mu.Unlock()

	log.Println("Server exiting")
}
