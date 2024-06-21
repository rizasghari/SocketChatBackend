package handlers

import (
	"bytes"
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
	obsSocketModels "socketChat/internal/models/socket/observing"
	"socketChat/internal/msgs"
	"socketChat/internal/services"
	"socketChat/internal/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type SocketUserObservingHandler struct {
	mu          sync.Mutex
	ctx         context.Context
	upgrader    websocket.Upgrader
	hub         *obsSocketModels.SocketUserObservingHub
	authService *services.AuthenticationService
}

func NewSocketUserObservingHandler(
	redis *redis.Client,
	ctx context.Context,
	authService *services.AuthenticationService,
) *SocketUserObservingHandler {
	suoh := &SocketUserObservingHandler{
		ctx:         ctx,
		authService: authService,
		hub: &obsSocketModels.SocketUserObservingHub{
			Notifiers: make(map[uint][]*models.SocketClient),
			Mu:        sync.Mutex{},
			Redis:     redis,
		},
	}
	go suoh.handleRedisMessages()
	return suoh
}

func (suoh *SocketUserObservingHandler) HandleSocketUserObservingRoute(ctx *gin.Context) {
	// Authenticate user
	userInfo, err := suoh.authorize(ctx)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{err},
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
	defer func(ws *websocket.Conn) {
		err := ws.Close()
		if err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}(ws)

	// Set the user online status to online
	suoh.setOnlineStatus(userInfo.ID, true)

	// Subscribe to notifiers
	notifiers, err := suoh.retrieveNotifiersFromQuery(ctx)
	if err == nil && len(notifiers) > 0 {
		suoh.handleSubscription(ws, userInfo, notifiers)
	}

	// Keep socket alive to notify user
	suoh.keepSocketAlive(ws, userInfo.ID)
}

func (souh *SocketUserObservingHandler) authorize(ctx *gin.Context) (*models.Claims, error) {
	// Authenticate user
	jwtToken := ctx.Request.Header.Get("Authorization")
	if jwtToken == "" {
		return nil, errs.ErrUnauthorized
	}
	userInfo, err := utils.VerifyToken(jwtToken)
	if err != nil {
		return nil, err
	}
	return userInfo, nil
}

func (suoh *SocketUserObservingHandler) keepSocketAlive(ws *websocket.Conn, userId uint) {
	for {
		var buf bytes.Buffer
		err := ws.ReadJSON(&buf)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
				suoh.unsubscribe(userId)
				break
			}
			log.Printf("Error reading json from user %v: %v", userId, err)
			continue
		}
		log.Println("keepSocketAlive buf: ", buf.String())
	}
}

func (suoh *SocketUserObservingHandler) setOnlineStatus(userId uint, status bool) {
	// set user online status in db
	isOnline, lastSeen, err := suoh.authService.GetUserOnlineStatus(userId)
	if err != nil {
		log.Printf("Error while fetching user %v online status from db: %v", userId, err)
	}
	log.Printf("User %v previous online status: %v - last seen: %v", userId, isOnline, lastSeen)

	isOnline, lastSeen, err = suoh.authService.SetUserOnlineStatus(userId, status)
	if err != nil {
		log.Printf("failed to set user %v online status in db: %v", userId, err)
		return
	}
	log.Printf("User %v current online status: %v - last seen: %v", userId, isOnline, lastSeen)

	// Update user online status in cache
	err = suoh.updateUserOnlineStatusInCache(userId, status, *lastSeen)
	if err != nil {
		log.Printf("Error while updating user %v online status on cache: %v", userId, err)
	} else {
		status, lseen, err := suoh.fetchUserOnlineStatusFromCache(userId)
		if err != nil {
			log.Printf("Error while fetching user %v online status from cache: %v", userId, err)
		}
		log.Printf("User %v online status from cache: %v - %v", userId, status, lseen)
	}

	// Publish the new event to Redis
	redisEvent := obsSocketModels.ObservingSocketEvent{
		Event: enums.SOCKET_EVENT_NOTIFY,
		Payload: obsSocketModels.ObservingSocketPayload{
			UserId:     userId,
			IsOnline:   status,
			LastSeenAt: lastSeen,
		},
	}

	jsonEvent, err := json.Marshal(redisEvent)
	if err != nil {
		log.Println("failed to marshal jsonEvent: ", err)
		return
	}
	log.Println("setOnlineStatus jsonEvent: ", string(jsonEvent))
	if err := suoh.publish(suoh.hub.Redis, redisModels.REDIS_CHANNEL_OBSERVE, jsonEvent); err != nil {
		log.Println("failed to publish message: ", err)
		return
	}
}

func (suoh *SocketUserObservingHandler) updateUserOnlineStatusInCache(userID uint, status bool, lastSeen time.Time) error {
	expirationDuration := time.Duration(time.Hour * 24)

	// Save online status
	statusKey := fmt.Sprintf("user_online_status_%v", userID)
	var statusValue string
	if status {
		statusValue = "true"
	} else {
		statusValue = "false"
	}
	result := suoh.hub.Redis.Set(suoh.ctx, statusKey, statusValue, expirationDuration)
	if result.Err() != nil {
		return result.Err()
	}

	// Save last seen
	lastSeenKey := fmt.Sprintf("user_last_seen_%v", userID)
	result = suoh.hub.Redis.Set(suoh.ctx, lastSeenKey, lastSeen, expirationDuration)
	if result.Err() != nil {
		return result.Err()
	}

	return nil
}

func (suoh *SocketUserObservingHandler) fetchUserOnlineStatusFromCache(userID uint) (bool, *time.Time, error) {
	// Get online status
	statusKey := fmt.Sprintf("user_online_status_%v", userID)
	statusStr, err := suoh.hub.Redis.Get(suoh.ctx, statusKey).Result()
	if err != nil {
		return false, nil, err
	}
	var status bool
	if statusStr == "true" {
		status = true
	} else {
		status = false
	}

	// Get last seen
	lastSeenKey := fmt.Sprintf("user_last_seen_%v", userID)
	lastSeenStr, err := suoh.hub.Redis.Get(suoh.ctx, lastSeenKey).Result()
	if err != nil {
		return false, nil, err
	}
	lastSeen, err := utils.StrToTime(lastSeenStr)
	if err != nil {
		return false, nil, err
	}

	return status, lastSeen, nil
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
	log.Println("retrieveNotifiersFromQuery notifiers: ", notifiers)
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
	return ws, nil
}

func (suoh *SocketUserObservingHandler) handleSubscription(ws *websocket.Conn, userInfo *models.Claims, notifiers []uint) {
	observer := &models.SocketClient{
		Conn:   ws,
		UserId: userInfo.ID,
	}
	suoh.subscribe(observer, notifiers)
	suoh.handleDisconnection(observer)
}

func (suoh *SocketUserObservingHandler) handleDisconnection(observer *models.SocketClient) {
	observer.Conn.SetCloseHandler(func(code int, text string) error {
		suoh.unsubscribe(observer.UserId)
		return nil
	})
}

func (suoh *SocketUserObservingHandler) subscribe(observer *models.SocketClient, notifiersToObserve []uint) {
	suoh.mu.Lock()
	defer suoh.mu.Unlock()
	for _, notifier := range notifiersToObserve {
		// Add Notifier to observing hub if not exists
		if _, exists := suoh.hub.Notifiers[notifier]; !exists {
			suoh.hub.Notifiers[notifier] = []*models.SocketClient{}
		}
		// Add observer to notifier if not observing yet and save it in redis cache
		if observing := slices.Contains(suoh.hub.Notifiers[notifier],
			&models.SocketClient{Conn: observer.Conn, UserId: observer.UserId}); !observing {
			err := suoh.saveObserverNotifiersInCache(observer.UserId, notifier)
			if err != nil {
				log.Fatalf("Could not add the notifier to observer notifiers in cache: %v", err)
				return
			}
			suoh.hub.Notifiers[notifier] = append(suoh.hub.Notifiers[notifier], observer)
		}
	}
}

func (suoh *SocketUserObservingHandler) unsubscribe(observer uint) {
	suoh.mu.Lock()
	defer suoh.mu.Unlock()
	// set the observer offline
	suoh.setOnlineStatus(observer, false)

	// Fetch observer notifiers from cache
	notifiers, err := suoh.fetchObserverNotifiersFromCache(observer)
	if err != nil {
		log.Printf("Could not fetch observer notifiers from cache: %v", err)
		return
	}
	if len(notifiers) == 0 {
		return
	}
	log.Printf("unsubscribe - fetchObserverNotifiersFromCache for observer %v: %v", observer, notifiers)

	// Remove observer from redis cache
	err = suoh.hub.Redis.Del(suoh.ctx, fmt.Sprintf("observer_notifiers_%d", observer)).Err()
	if err != nil {
		log.Printf("Could not remove observer from redis cache: %v", err)
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
		log.Println("handleRedisMessages New redis message received")
		suoh.send(redisMessage)
	}
}

func (suoh *SocketUserObservingHandler) send(redisMessage obsSocketModels.ObservingSocketEvent) {
	log.Printf("Sending message to notifier observers. Notifier: %v", redisMessage.Payload.UserId)
	suoh.mu.Lock()
	defer suoh.mu.Unlock()
	if notifier, ok := suoh.hub.Notifiers[redisMessage.Payload.UserId]; ok {
		log.Printf("Found notifier %v", redisMessage.Payload.UserId)
		if len(notifier) > 0 {
			for _, client := range notifier {
				log.Printf("Found observer %v", client.UserId)
				if err := client.Conn.WriteJSON(redisMessage); err != nil {
					log.Printf("Error writing json: %v", err)
					err := client.Conn.Close()
					if err != nil {
						return
					}
					suoh.unsubscribe(client.UserId)
				}
			}
		} else {
			log.Printf("Notifier %v doesn't have any subscribed observers", redisMessage.Payload.UserId)
		}
	} else {
		log.Printf("No notifier found with %v key", redisMessage.Payload.UserId)
	}
}

func (suoh *SocketUserObservingHandler) publish(redis *redis.Client, channel string, message []byte) error {
	log.Printf("Publishing message to channel %v with message %v", channel, string(message))
	return redis.Publish(suoh.ctx, channel, message).Err()
}

func (suoh *SocketUserObservingHandler) subscribeToChannel(redis *redis.Client, channel string) <-chan *redis.Message {
	log.Printf("Subscribing to redis channel %v", channel)
	pubsub := redis.Subscribe(suoh.ctx, channel)
	_, err := pubsub.Receive(suoh.ctx)
	if err != nil {
		log.Fatalf("Could not subscribe to channel: %v", err)
	}
	return pubsub.Channel()
}
