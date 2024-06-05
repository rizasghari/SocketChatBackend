package repositories

import (
	"socketChat/internal/models"

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
