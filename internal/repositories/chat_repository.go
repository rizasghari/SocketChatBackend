package repositories

import (
	"socketChat/internal/models"
	"socketChat/internal/utils"

	"gorm.io/gorm"
)

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{
		db: db,
	}
}

func (chr *ChatRepository) CreateConversation(conversationData *models.CreateConversationRequestBody) (*models.Conversation, []error) {
	var errors []error

	conversation := &models.Conversation{
		Type: conversationData.Type,
		Name: conversationData.Name,
	}

	err := chr.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conversation).Error; err != nil {
			// return any error will rollback
			return err
		}

		for _, userId := range conversationData.Users {
			err := tx.Create(&models.ConversationMember{
				ConversationID: conversation.ID,
				UserID:         userId,
			}).Error

			if err != nil {
				// return any error will rollback
				return err
			}
		}

		// return nil will commit the whole transaction
		return nil
	})

	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	return conversation, nil
}

func (chr *ChatRepository) GetUserConversations(userID uint, page, size int) (*models.ConversationListResponse, []error) {
	var errors []error
	var conversations []models.Conversation
	var conversationResponses []models.ConversationResponse
	var total int64

	transactionErr := chr.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Scopes(utils.Paginate(page, size)).
			Preload("Members").
			Where("id IN (SELECT conversation_id FROM conversation_members WHERE user_id = ?)", userID).
			Find(&conversations).Error; err != nil {
			return err
		}

		if err := tx.
			Model(&models.Conversation{}).
			Where("id IN (SELECT conversation_id FROM conversation_members WHERE user_id = ?)", userID).
			Count(&total).Error; err != nil {
			return err
		}

		return nil
	})
	if transactionErr != nil {
		errors = append(errors, transactionErr)
		return nil, errors
	}

	for _, conversation := range conversations {
		conversationResponses = append(conversationResponses, conversation.ToConversationResponse())

	}

	return &models.ConversationListResponse{
		Conversations: conversationResponses,
		Page:          page,
		Size:          size,
		Total:         total,
	}, nil
}

func (chr *ChatRepository) SendMessage(message *models.Message) (*models.Message, []error) {
	var errors []error
	if err := chr.db.Create(message).Error; err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	return message, nil
}

func (chr *ChatRepository) GetMessagesByConversationId(conversationID uint, page, size int) (*models.MessageListResponse, []error) {
	var errors []error
	var messages []models.Message
	var total int64

	transactionErr := chr.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Scopes(utils.Paginate(page, size)).
			Where("conversation_id = ?", conversationID).
			Order("created_at DESC").
			Find(&messages).Error; err != nil {
			return err
		}

		if err := tx.
			Model(&models.Message{}).
			Where("conversation_id = ?", conversationID).
			Count(&total).Error; err != nil {
			return err
		}

		return nil
	})
	if transactionErr != nil {
		errors = append(errors, transactionErr)
		return nil, errors
	}

	return &models.MessageListResponse{
		Messages: messages,
		Page:     page,
		Size:     size,
		Total:    total,
	}, nil
}
