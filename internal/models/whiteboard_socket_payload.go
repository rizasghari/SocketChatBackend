package models

type WhiteboardSocketPayload struct {
	DrawnID      uint   `json:"drawn_id"`
	Drawer       uint   `json:"drawer_user_id"`
	Points       Points `json:"points"`
	Paint        Paint  `json:"paint"`
	WhiteboardId uint   `json:"whiteboard_id"`
}
