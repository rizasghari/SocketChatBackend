package models

import "time"

type UserResponse struct {
	ID           uint       `json:"id"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	ProfilePhoto *string    `json:"profile_photo"`
	IsOnline     bool       `json:"is_online"`
	LastSeen     *time.Time `json:"last_seen"`
}
