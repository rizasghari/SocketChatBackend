package observing

import "time"

type ObservingSocketPayload struct {
	UserId   uint       `json:"user_id"`
	IsOnline bool       `json:"is_online"`
	LastSeenAt *time.Time `json:"last_seen_at"`
}
