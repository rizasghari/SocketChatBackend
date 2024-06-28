package models

type WhiteboardSocketPayload struct {
	ID           uint    `json:"id"`
	Drawer       uint    `json:"drawer_user_id"`
	Points       *Points `json:"points"`
	WhiteboardId uint    `json:"whiteboard_id"`
}
