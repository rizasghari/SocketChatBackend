package services

import (
	"log"
	"socketChat/internal/models"
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

func (ws *WhiteboardService) CreateNewWhiteboard(whiteboard *models.Whiteboard) (*models.Whiteboard, error) {
	// Check if a whiteboard already has been created before for the conversation
	found, err := ws.whiteboardRepo.FindConversationWhiteboard(whiteboard.ConversationID)
	if (err != nil) {
		log.Printf("CreateNewWhiteboard / FindConversationWhiteboard / error: %v", err)
		return ws.whiteboardRepo.CreateNewWhiteboard(whiteboard)
	}
	return found, nil
}


