package services

import (
	"socketChat/internal/models"
	"socketChat/internal/repositories"
	"socketChat/internal/errs"
)

type ChatService struct {
	chatRepo *repositories.ChatRepository
}

func NewChatService(chatRepo *repositories.ChatRepository) *ChatService {
	return &ChatService{
		chatRepo: chatRepo,
	}
}

func (cs *ChatService) CreateConversation(conversationData *models.CreateConversationRequestBody) (*models.ConversationResponse, []error) {
	existingConversationId, errs := cs.chatRepo.FindConversationBetweenTwoUsers(conversationData.Users[0], conversationData.Users[1])
	if len(errs) > 0 {
		return nil, errs
	}
	if existingConversationId > 0 {
		return cs.chatRepo.GetConversationById(existingConversationId)
	}
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

func (cs *ChatService) SeenMessage(messageIds []uint, seenerId uint) []error {
	// Validate message id
	if len(messageIds) <= 0 {
		return []error{errs.ErrMessageNotFound}
	}
	return cs.chatRepo.SeenMessage(messageIds, seenerId)
}

func (cs *ChatService) GetConversationUnReadMessagesForUser(conversationID, userID uint) (int, error) {
	return cs.chatRepo.GetConversationUnReadMessagesForUser(conversationID, userID)
}

func (cs *ChatService) GetUsersWhoHaveSentMessage(concurrent bool) ([]*models.UserResponse, error) {
	if concurrent {
		return cs.chatRepo.GetUsersWhoHaveSentMessageConcurrent()
	} else {
		return cs.chatRepo.GetUsersWhoHaveSentMessage()
	}
}