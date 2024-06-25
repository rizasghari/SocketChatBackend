package services

import (
	"socketChat/internal/models/whiteboard"
	"socketChat/internal/repositories"
)

type WhiteboardService struct {
	whiteboardRepo *repositories.WhiteboardRepository
}

func NewWhiteboardService(whiteboardRepo *repositories.WhiteboardRepository) *WhiteboardService {
	return &WhiteboardService{
		whiteboardRepo: whiteboardRepo,
	}
}

func (ws *WhiteboardService) CreateNewWhiteboard(whiteboard *whiteboard.Whiteboard) error {
	// TODO: validate whiteboard
	return ws.whiteboardRepo.CreateNewWhiteboard(whiteboard)
}


