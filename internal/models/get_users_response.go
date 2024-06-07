package models

type GetUsersResponse struct {
	Users []User `json:"users"`
	Page  int    `json:"page"`
	Size  int    `json:"size"`
	Total int64  `json:"total"`
}
