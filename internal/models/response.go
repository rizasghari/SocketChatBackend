package models

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Errors  []error     `json:"errors"`
	Data    interface{} `json:"data"`
}
