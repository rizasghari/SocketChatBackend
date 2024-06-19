package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"socketChat/internal/enums"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	"socketChat/internal/models/socket/observing"
	"socketChat/internal/msgs"
	"socketChat/internal/utils"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type SocketUserObservingHandler struct {
	mu       sync.Mutex
	ctx      context.Context
	upgrader websocket.Upgrader
	hub      *observing.SocketUserObservingHub
}

func NewSocketUserObservingHandler(ctx context.Context, redis *redis.Client) *SocketUserObservingHandler {
	return &SocketUserObservingHandler{
		ctx: ctx,
		hub: &observing.SocketUserObservingHub{
			Notifiers: make(map[uint][]*models.SocketClient),
			Mu:        sync.Mutex{},
			Redis:     nil,
		},
	}
}

func (suoh *SocketUserObservingHandler) HandleSocketUserObservingRoute(ctx *gin.Context) {
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

	suoh.HandleConnections(ctx, userInfo)
}

func (suoh *SocketUserObservingHandler) StartUserObservingSocket() {
	suoh.InitializeSocketUpgrader()
	go suoh.HandleRedisMessages()
}

func (suoh *SocketUserObservingHandler) InitializeSocketUpgrader() {
	suoh.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func (suoh *SocketUserObservingHandler) HandleConnections(ctx *gin.Context, userInfo *models.Claims) {
	ws, err := suoh.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
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
	suoh.handleDiconnectedClient(ws, userInfo.ID)

	// Add client to hub
	suoh.handleConversationAndClinet(userInfo.ID, ws)

	// Handle incoming messages
	suoh.handleIncommingMessagesWithEvent(ws, userInfo)
}

func (sch *SocketUserObservingHandler) handleDiconnectedClient(ws *websocket.Conn, userId uint) {
	ws.SetCloseHandler(func(code int, text string) error {
		sch.deleteDiconnectedClientFromConversation(userId)
		return nil
	})
}

func (suoh *SocketUserObservingHandler) subscribeObserverToNotifiers(observer uint, notifiersToObserve []uint, ws *websocket.Conn) {
	suoh.mu.Lock()
	defer suoh.mu.Unlock()
	for _, notifier := range notifiersToObserve {
		// Add Notifier to observing hub if not exists
		if _, exists := suoh.hub.Notifiers[notifier]; !exists {
			suoh.hub.Notifiers[notifier] = []*models.SocketClient{}
		}
		// Add observer to notifier if not observing yet
		if isAlreadyObserving := slices.Contains(suoh.hub.Notifiers[notifier], &models.SocketClient{Conn: ws, UserId: observer}); !isAlreadyObserving {
			suoh.hub.Notifiers[notifier] = append(suoh.hub.Notifiers[notifier],
				&models.SocketClient{
					Conn:   ws,
					UserId: observer,
				},
			)
			err := suoh.saveObserverNotifiersInCache(observer, notifier)
			if err != nil {
				log.Fatalf("Could not save the slice: %v", err)
				return
			}
		}
	}
}

func (suoh *SocketUserObservingHandler) saveObserverNotifiersInCache(observerUserId uint, notifierToObserve uint) error {
	notifierStr := fmt.Sprintf("%d", notifierToObserve)
	key := fmt.Sprintf("observer-%d", observerUserId)
	err := suoh.hub.Redis.RPush(suoh.ctx, key, notifierStr).Err()
	if err != nil {
		log.Fatalf("Could not save the slice: %v", err)
		return err
	}
	return nil
}

func (sch *SocketUserObservingHandler) handleIncommingMessagesWithEvent(ws *websocket.Conn, userInfo *models.Claims, conversationId uint) {
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
			//
		default:
			log.Printf("Unknown event: %v", event)
		}
	}
}

func (suoh *SocketUserObservingHandler) HandleRedisMessages() {
	ch := suoh.SubscribeToChannel(suoh.hub.Redis, "chat_channel")
	for msg := range ch {
		var redisMessage redisModels.RedisPublishedMessage
		if err := json.Unmarshal([]byte(msg.Payload), &redisMessage); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}
		suoh.SendMessageToClient(redisMessage)
	}
}

func (sch *SocketUserObservingHandler) unSubscribeObserverFromNotifiers(userId uint) {
	sch.mu.Lock()
	defer sch.mu.Unlock()
	// Remove disconnected client from conversation
	for i, client := range sch.hub.Notifiers[userId] {
		if client.UserId == userId {
			sch.hub.Notifiers[userId] = append(sch.hub.Notifiers[userId][:i], sch.hub.Notifiers[userId][i+1:]...)
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

func (sch *SocketUserObservingHandler) HandleRedisMessages() {
	ch := sch.SubscribeToChannel(sch.hub.Redis, "chat_channel")
	for msg := range ch {
		var redisMessage redisModels.RedisPublishedMessage
		if err := json.Unmarshal([]byte(msg.Payload), &redisMessage); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}
		sch.SendMessageToClient(redisMessage)
	}
}

func (sch *SocketUserObservingHandler) SendMessageToClient(redisMessage redisModels.RedisPublishedMessage) {
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

func (sch *SocketUserObservingHandler) PublishMessage(redis *redis.Client, channel string, message []byte) error {
	return redis.Publish(sch.ctx, channel, message).Err()
}

func (sch *SocketUserObservingHandler) SubscribeToChannel(redis *redis.Client, channel string) <-chan *redis.Message {
	pubsub := redis.Subscribe(sch.ctx, channel)
	_, err := pubsub.Receive(sch.ctx)
	if err != nil {
		log.Fatalf("Could not subscribe to channel: %v", err)
	}
	return pubsub.Channel()
}
