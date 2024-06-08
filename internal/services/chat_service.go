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

func (cs *ChatService) CreateConversation(conversationData *models.CreateConversationRequestBody) (*models.Conversation, []error) {
	// TODO: Add conversation validation
	return cs.chatRepo.CreateConversation(conversationData)
}

func (cs *ChatService) GetUserConversations(userID uint, page, size int) (*models.ConversationListResponse, []error) {
	return cs.chatRepo.GetUserConversations(userID, page, size)
}

func (cs *ChatService) SaveMessage(message *models.Message) (*models.Message, []error) {
	return cs.chatRepo.SaveMessage(message)
}

func (cs *ChatService) GetMessagesByConversationId(conversationID uint, page, size int) (*models.MessageListResponse, []error) {
	return cs.chatRepo.GetMessagesByConversationId(conversationID, page, size)
}

func (cs *ChatService) CheckConversationExists(conversationID uint) bool {
	return cs.chatRepo.CheckConversationExists(conversationID)
}

func (cs *ChatService) CheckUserInConversation(userID, conversationID uint) bool {
	return cs.chatRepo.CheckUserInConversation(userID, conversationID)
}