package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"socketChat/internal/enums"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	redisModels "socketChat/internal/models/redis"
	"socketChat/internal/models/socket/observing"
	obsSocketModels "socketChat/internal/models/socket/observing"
	"socketChat/internal/msgs"
	"socketChat/internal/services"
	"socketChat/internal/utils"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type SocketUserObservingHandler struct {
	mu          sync.Mutex
	ctx         context.Context
	upgrader    websocket.Upgrader
	hub         *observing.SocketUserObservingHub
	authService *services.AuthenticationService
}

func NewSocketUserObservingHandler(redis *redis.Client, ctx context.Context, authService *services.AuthenticationService) *SocketUserObservingHandler {
	suoh := &SocketUserObservingHandler{
		ctx: ctx,
		authService: authService,
		hub: &observing.SocketUserObservingHub{
			Notifiers: make(map[uint][]*models.SocketClient),
			Mu:        sync.Mutex{},
			Redis:     nil,
		},
	}
	go suoh.handleRedisMessages()
	return suoh
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

	// Handle operation
	switch operation {
	case enums.OBS_SOCK_OP_SUBSCRIBE:
		notifiers, err := suoh.retrieveNotifiersFromQuery(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
				Success: false,
				Message: msgs.MsgOperationFailed,
				Errors:  []error{err},
			})
		}
		suoh.handleSubsciption(ctx, ws, userInfo, notifiers)
	case enums.OBS_SOCK_OP_SET_STATUS:
		status, err := suoh.retrieveStatusFromQuery(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
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
	isOnline := strings.ToLower(status) == "online"

	// set user online status in db

	// Publish the new event to Redis
	redisEvent := obsSocketModels.ObservingSocketEvent{
		Event: enums.OBS_SOCK_OP_NOTIFY,
		Payload: obsSocketModels.ObservingSocketPayload{
			UserId:   userInfo.ID,
			IsOnline: isOnline,
			LastSeen: nil,
		},
	}

	jsonEvent, err := json.Marshal(redisEvent)
	if err != nil {
		log.Println("failed to marshal jsonEvent: ", err)
		return
	}
	log.Println("jsonEvent: ", string(jsonEvent))
	if err := suoh.publishMessage(suoh.hub.Redis, redisModels.REDIS_CHANNEL_OBSERVE, jsonEvent); err != nil {
		log.Println("failed to publish message: ", err)
		return
	}
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
	observer := &models.SocketClient{
		Conn:   ws,
		UserId: userInfo.ID,
	}

	// Add observer to observing notifiers and subscribe to hub
	suoh.subscribeObserverToNotifiers(observer, notifiers)

	// unsubscribe from hub and notifiers if user disconnects
	suoh.setObserverDisconnectionListener(observer)
}

func (sch *SocketUserObservingHandler) setObserverDisconnectionListener(observer *models.SocketClient) {
	observer.Conn.SetCloseHandler(func(code int, text string) error {
		sch.unsubscribeObserverFromNotifiers(observer.UserId)
		return nil
	})
}

func (suoh *SocketUserObservingHandler) subscribeObserverToNotifiers(observer *models.SocketClient, notifiersToObserve []uint) {
	suoh.mu.Lock()
	defer suoh.mu.Unlock()
	for _, notifier := range notifiersToObserve {
		// Add Notifier to observing hub if not exists
		if _, exists := suoh.hub.Notifiers[notifier]; !exists {
			suoh.hub.Notifiers[notifier] = []*models.SocketClient{}
		}
		// Add observer to notifier if not observing yet and save it in redis cache
		if observing := slices.Contains(suoh.hub.Notifiers[notifier], &models.SocketClient{Conn: observer.Conn, UserId: observer.UserId}); !observing {
			err := suoh.saveObserverNotifiersInCache(observer.UserId, notifier)
			if err != nil {
				log.Fatalf("Could not add the notifier to observer notifiers in cache: %v", err)
				return
			}
			suoh.hub.Notifiers[notifier] = append(suoh.hub.Notifiers[notifier], observer)
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

func (suoh *SocketUserObservingHandler) handleRedisMessages() {
	ch := suoh.subscribeToChannel(suoh.hub.Redis, redisModels.REDIS_CHANNEL_OBSERVE)
	for msg := range ch {
		var redisMessage obsSocketModels.ObservingSocketEvent
		if err := json.Unmarshal([]byte(msg.Payload), &redisMessage); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}
		suoh.sendMessageToClient(redisMessage)
	}
}

func (suoh *SocketUserObservingHandler) sendMessageToClient(redisMessage obsSocketModels.ObservingSocketEvent) {
	suoh.mu.Lock()
	defer suoh.mu.Unlock()

	if notifier, ok := suoh.hub.Notifiers[redisMessage.Payload.UserId]; ok {
		for _, client := range notifier {
			if err := client.Conn.WriteJSON(redisMessage); err != nil {
				log.Printf("Error writing json: %v", err)
				err := client.Conn.Close()
				if err != nil {
					return
				}
				suoh.unsubscribeObserverFromNotifiers(client.UserId)
			}
		}
	}
}

func (suoh *SocketUserObservingHandler) publishMessage(redis *redis.Client, channel string, message []byte) error {
	return redis.Publish(suoh.ctx, channel, message).Err()
}

func (suoh *SocketUserObservingHandler) subscribeToChannel(redis *redis.Client, channel string) <-chan *redis.Message {
	pubsub := redis.Subscribe(suoh.ctx, channel)
	_, err := pubsub.Receive(suoh.ctx)
	if err != nil {
		log.Fatalf("Could not subscribe to channel: %v", err)
	}
	return pubsub.Channel()
}
