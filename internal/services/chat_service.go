package services

import (
	"socketChat/internal/models"
	"socketChat/internal/repositories"
)

type ChatService struct {
	chatRepo *repositories.ChatRepository
}

func NewChatService(chatRepo *repositories.ChatRepository) *ChatService {
	return &ChatService{
		chatRepo: chatRepo,
	}
}

func (cs *ChatService) CreateConversation(conversation *models.CreateConversationRequestBody) (*models.Conversation, []error) {
	// TODO: Add conversation validation
	return cs.chatRepo.CreateConversation(conversation)
}