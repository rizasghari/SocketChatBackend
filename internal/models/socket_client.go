package models

import "github.com/gorilla/websocket"

type SocketClient struct {
	Conn           *websocket.Conn
	ConversationID uint
}
