package models

type ProfileResponse struct {
	ID           uint    `json:"id"`
	Email        string  `json:"email"`
	FirstName    string  `json:"first_name"`
	LastName     string  `json:"last_name"`
	ProfilePhoto *string `json:"profile_photo"`
}
