package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"socketChat/internal/enums"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	socketModels "socketChat/internal/models/socket"
	"socketChat/internal/models/socket/observing"
	"socketChat/internal/msgs"
	"socketChat/internal/utils"
	"strconv"
	"strings"
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

	// Get operation
	operation := ctx.Param("operation")
	if operation == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrObservingSocketOperationRequired},
		})
		return
	}

	// Upgrade HTTP connection to WebSocket
	ws, err := suoh.upgradeHttpToWs(ctx)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{err},
		})
		return
	}
	defer ws.Close()

	switch operation {
	case enums.OBS_SOCK_OP_SUBSCRIBE:
		notifiers, err := suoh.retrieveNotifiersFromQuery(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
				Success: false,
				Message: msgs.MsgOperationFailed,
				Errors:  []error{err},
			})
		}
		suoh.handleSubsciption(ctx, ws, userInfo, notifiers)
	case enums.OB_SOCK_OP_SET_STATUS:
		status, err := suoh.retrieveStatusFromQuery(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
				Success: false,
				Message: msgs.MsgOperationFailed,
				Errors:  []error{err},
			})
		}
		suoh.handleSetOnlineStatus(ctx, ws, userInfo, status)
	default:
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrInvalidObservingSocketOperation},
		})
		return
	}
}

func (souh *SocketUserObservingHandler) retrieveStatusFromQuery(ctx *gin.Context) (string, error) {
	statusQuery := ctx.Query("status")
	if statusQuery == "" {
		return "", errs.ErrObservingSocketStatusRequired
	}
	return statusQuery, nil
}

func (suoh *SocketUserObservingHandler) handleSetOnlineStatus(ctx *gin.Context, ws *websocket.Conn, userInfo *models.Claims, status string) {
	// set user online status in db

	// notify all observers
}

func (suoh *SocketUserObservingHandler) retrieveNotifiersFromQuery(ctx *gin.Context) ([]uint, error) {
	notifiersQuery := ctx.Query("notifiers")
	if notifiersQuery == "" {
		return []uint{}, errs.ErrInvalidRequest
	}
	strNotifiers := strings.Split(notifiersQuery, ",")
	var notifiers []uint
	for _, strNum := range strNotifiers {
		num, err := strconv.Atoi(strNum)
		if err != nil {
			return []uint{}, errs.ErrInvalidRequest
		}
		notifiers = append(notifiers, uint(num))
	}
	return notifiers, nil
}

func (suoh *SocketUserObservingHandler) StartUserObservingSocket() {
	go suoh.HandleRedisMessages()
}

func (suoh *SocketUserObservingHandler) upgradeHttpToWs(ctx *gin.Context) (*websocket.Conn, error) {
	suoh.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	ws, err := suoh.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return nil, err
	}
	defer func(ws *websocket.Conn) {
		err := ws.Close()
		if err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}(ws)
	return ws, nil
}

func (suoh *SocketUserObservingHandler) handleSubsciption(ctx *gin.Context, ws *websocket.Conn, userInfo *models.Claims, notifiers []uint) {

	// unsubscribe from hub and notifiers if user disconnects
	suoh.setObserverDisconnectedListener(&models.SocketClient{Conn: ws, UserId: userInfo.ID})

	// Add observer to observing notifiers and subscribe to hub
	suoh.subscribeObserverToNotifiers(userInfo.ID, notifiers, ws)

	// Handle incoming messages
	suoh.handleIncommingMessagesWithEvent(ws, userInfo)
}

func (sch *SocketUserObservingHandler) setObserverDisconnectedListener(observer *models.SocketClient) {
	observer.Conn.SetCloseHandler(func(code int, text string) error {
		sch.unsubscribeObserverFromNotifiers(observer.UserId)
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
		// Add observer to notifier if not observing yet and save it in redis cache
		if observing := slices.Contains(suoh.hub.Notifiers[notifier], &models.SocketClient{Conn: ws, UserId: observer}); !observing {
			err := suoh.saveObserverNotifiersInCache(observer, notifier)
			if err != nil {
				log.Fatalf("Could not add the notifier to observer notifiers in cache: %v", err)
				return
			}
			suoh.hub.Notifiers[notifier] = append(suoh.hub.Notifiers[notifier],
				&models.SocketClient{
					Conn:   ws,
					UserId: observer,
				},
			)
		}
	}
}

func (suoh *SocketUserObservingHandler) unsubscribeObserverFromNotifiers(observer uint) {
	suoh.mu.Lock()
	defer suoh.mu.Unlock()

	// Fetch observer notifiers from cache
	notifiers, err := suoh.fetchObserverNotifiersFromCache(observer)
	if err != nil {
		log.Println("Could not fetch observer notifiers from cache: %v", err)
		return
	}
	if len(notifiers) == 0 {
		return
	}

	// Remove observer from redis cache
	err = suoh.hub.Redis.Del(suoh.ctx, fmt.Sprintf("observer_notifiers_%d", observer)).Err()
	if err != nil {
		log.Println("Could not remove observer from redis cache: %v", err)
		return
	}

	// Remove observer from notifiers
	for _, notifier := range notifiers {
		for i, client := range suoh.hub.Notifiers[notifier] {
			if client.UserId == observer {
				suoh.hub.Notifiers[notifier] = append(suoh.hub.Notifiers[notifier][:i], suoh.hub.Notifiers[notifier][i+1:]...)
				break
			}
		}
		// Check if the notifier observers is empty and remove it from the hub
		if len(suoh.hub.Notifiers[notifier]) == 0 {
			delete(suoh.hub.Notifiers, notifier)
		}
	}
}

func (suoh *SocketUserObservingHandler) saveObserverNotifiersInCache(observer uint, notifier uint) error {
	key := fmt.Sprintf("observer_notifiers_%d", observer)
	err := suoh.hub.Redis.RPush(suoh.ctx, key, fmt.Sprintf("%d", notifier)).Err()
	if err != nil {
		return err
	}
	return nil
}

func (souh *SocketUserObservingHandler) fetchObserverNotifiersFromCache(observer uint) ([]uint, error) {
	key := fmt.Sprintf("observer_notifiers_%d", observer)
	value, err := souh.hub.Redis.LRange(souh.ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	notifiers := make([]uint, len(value))
	for i, str := range value {
		notifier, err := strconv.ParseUint(str, 10, 32)
		if err != nil {
			return nil, err
		}
		notifiers[i] = uint(notifier)
	}
	return notifiers, nil
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
