package models

type CreateDrawnRequest struct {
	Drawer         uint    `json:"drawer_user_id"`
	Points         *Points `json:"points"`
	WhiteboardId   uint    `json:"whiteboard_id"`
	ConversationID uint    `json:"conversation_idß"`
}
