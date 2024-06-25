package models

type SocketWhiteboardHub struct {
	// [whiteboard_id] => []*SocketClient
	Whiteboards map[uint][]*SocketClient
}
