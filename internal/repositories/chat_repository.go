package repositories

import (
	"socketChat/internal/errs"
	"socketChat/internal/models"
	"socketChat/internal/utils"
	"time"

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

func (chr *ChatRepository) CreateConversation(conversationData *models.CreateConversationRequestBody) (*models.ConversationResponse, []error) {
	var errors []error

	conversation := models.Conversation{
		Type: conversationData.Type,
	}

	err := chr.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&conversation).Error; err != nil {
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

	conversationResponse := conversation.ToConversationResponse()

	return &conversationResponse, nil
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
			Order("updated_at DESC").
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

func (chr *ChatRepository) SaveMessage(message *models.Message) (*models.Message, []error) {
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

func (chr *ChatRepository) CheckConversationExists(conversationID uint) bool {
	var count int64
	chr.db.Model(&models.Conversation{}).Where("id = ?", conversationID).Count(&count)
	return count > 0
}

func (chr *ChatRepository) CheckUserInConversation(userID, conversationID uint) bool {
	var count int64
	chr.db.Model(&models.ConversationMember{}).Where("user_id = ? AND conversation_id = ?", userID, conversationID).Count(&count)
	return count > 0
}

func (chr *ChatRepository) SeenMessage(messageId, seenerId uint) []error {
	var errors []error
	// Update if not seen yet and sender is not the seener to prevent message owner from marking it as seen
	result := chr.db.Model(&models.Message{}).Where("id = ? AND seen_at IS NULL AND sender_id != ?", messageId, seenerId).Update("seen_at", time.Now())
	if err := result.Error; err != nil {
		errors = append(errors, err)
		return errors
	}
	if result.RowsAffected == 0 {
		errors = append(errors, errs.ErrMessageNotFound)
		return errors
	}
	return nil
}

func (chr *ChatRepository) FindConversationBetweenTwoUsers(userID1, userID2 uint) (uint, []error) {
	var errors []error

	rows, err := chr.db.Table("conversation_members AS cm1").
		Select("cm1.conversation_id").
		Joins("INNER JOIN conversation_members AS cm2 ON cm1.conversation_id = cm2.conversation_id").
		Where("cm1.user_id = ? AND cm2.user_id = ?", userID1, userID2).
		Rows()

	if err != nil {
		errors = append(errors, err)
		return 0, errors
	}
	defer rows.Close()

	var conversationID uint
	for rows.Next() {
		if err := rows.Scan(&conversationID); err != nil {
			errors = append(errors, err)
			return 0, errors
		}
	}
	if err := rows.Err(); err != nil {
		errors = append(errors, err)
		return 0, errors
	}

	return conversationID, nil
}

func (chr *ChatRepository) GetConversationById(conversationID uint) (*models.ConversationResponse, []error) {
	var errors []error
	var conversation models.Conversation
	result := chr.db.Where("id = ?", conversationID).First(&conversation)
	if err := result.Error; err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	if result.RowsAffected == 0 {
		errors = append(errors, errs.ErrConversationNotFound)
		return nil, errors
	}
	conversationResponse := conversation.ToConversationResponse()
	return &conversationResponse, nil
}
