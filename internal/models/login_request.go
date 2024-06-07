package models

type LoginRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
