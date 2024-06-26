package models

type WhiteboardSocketPayload struct {
	Drawer       uint    `json:"drawer_user_id"`
	Points       *Points `json:"points"`
	WhiteboardId uint    `json:"whiteboard_id"`
}
