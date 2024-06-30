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
	if err != nil {
		return ws.whiteboardRepo.CreateNewWhiteboard(whiteboard)
	}
	return found, nil
}

func (ws *WhiteboardService) CreareNewDrawn(drawn *models.Drawn) (*models.Drawn, error) {
	// Check if a drawn already has been created before for the whiteboard and user
	found, err := ws.whiteboardRepo.FindWhiteboardDrawn(drawn.WhiteboardID, drawn.Drawer)
	if err != nil {
		log.Printf("CreareNewDrawn / FindWhiteboardDrawn / error: %v", err)
		return ws.whiteboardRepo.CreateNewDrawn(drawn)
	}
	return found, nil
}

func (ws *WhiteboardService) CreateSubDrawn(subDrownPayload *models.WhiteboardSocketPayload) (*models.SubDrawn, error) {
	return ws.whiteboardRepo.CreateSubDrawn(subDrownPayload)
}
