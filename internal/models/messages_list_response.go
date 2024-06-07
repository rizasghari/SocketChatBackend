package models

type MessageListResponse struct {
	Messages []Message `json:"messages"`
	Page     int       `json:"page"`
	Size     int       `json:"size"`
	Total    int64     `json:"total"`
}
