package repositories

import (
	"log"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	"socketChat/internal/utils"
	"sync"
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

	conversationFromDB, errs := chr.GetConversationById(conversation.ID)

	if len(errs) > 0 {
		return nil, errs
	}

	return conversationFromDB, nil
}

func (chr *ChatRepository) GetUserConversations(userID uint, page, size int) (*models.ConversationListResponse, []error) {
	var errors []error
	var conversations []models.Conversation
	var conversationResponses []models.ConversationResponse
	var total int64

	transactionErr := chr.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Scopes(utils.Paginate(page, size)).
			Preload("Members"). // The name of filed in conversation struct
			Preload("Whiteboard"). // The name of filed in conversation struct
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
		lastMessaege, _ := chr.GetConversationLastMessage(conversation.ID)
		unread, err := chr.GetConversationUnReadMessagesForUser(conversation.ID, userID)
		if err != nil {
			errors = append(errors, err)
			return nil, errors
		}
		conversationResponses = append(conversationResponses, conversation.ToConversationResponse(lastMessaege, unread))
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
	transactionErr := chr.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(message).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Conversation{}).
			Where("id = ?", message.ConversationID).
			Update("updated_at", time.Now()).Error; err != nil {
			return err
		}
		return nil
	})
	if transactionErr != nil {
		errors = append(errors, transactionErr)
		return nil, errors
	}
	return message, nil
}

func (chr *ChatRepository) GetConversationLastMessage(conversationID uint) (*models.Message, error) {
	var message models.Message
	if err := chr.db.
		Where("conversation_id = ?", conversationID).
		Last(&message).Error; err != nil {
		return nil, err
	}
	return &message, nil
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

func (chr *ChatRepository) SeenMessage(messageIds []uint, seenerId uint) []error {
	var errors []error
	// Update if not seen yet and sender is not the seener to prevent message owner from marking it as seen
	result := chr.db.Model(&models.Message{}).Where("id IN ? AND seen_at IS NULL AND sender_id != ?", messageIds, seenerId).Update("seen_at", time.Now())
	if err := result.Error; err != nil {
		errors = append(errors, err)
		return errors
	}
	if result.RowsAffected == 0 {
		errors = append(errors, errs.NoneOfMessagesSeen)
		return errors
	}
	return nil
}

func (chr *ChatRepository) GetConversationUnReadMessagesForUser(conversationID, userID uint) (int, error) {
	var count int = 0
	result := chr.db.Raw(
		"SELECT COUNT(*) FROM messages WHERE conversation_id = ? AND sender_id <> ? AND seen_at IS NULL",
		conversationID,
		userID,
	).Scan(&count)

	if err := result.Error; err != nil {
		return 0, err
	}

	return count, nil
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

	result := chr.db.
		Preload("Members").
		Preload("Whiteboard").
		Where("id = ?", conversationID).
		First(&conversation)

	if err := result.Error; err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	if result.RowsAffected == 0 {
		errors = append(errors, errs.ErrConversationNotFound)
		return nil, errors
	}
	lastMessaege, _ := chr.GetConversationLastMessage(conversation.ID)
	conversationResponse := conversation.ToConversationResponse(lastMessaege, 0)

	return &conversationResponse, nil
}

func (chr *ChatRepository) GetUsersWhoHaveSentMessageConcurrent() ([]*models.UserResponse, error) {
	var users []models.User
	wg := sync.WaitGroup{}
	start := time.Now()

	subQuery := chr.db.Raw("SELECT DISTINCT sender_id FROM messages")
	if err := chr.db.Raw("SELECT * FROM users WHERE id IN (?)", subQuery).Scan(&users).Error; err != nil {
		return nil, err
	}

	var userResponses = make([]*models.UserResponse, len(users))
	for index, user := range users {
		wg.Add(1)
		go func(user models.User, index int, wg *sync.WaitGroup) {
			defer wg.Done()
			userResponses[index] = user.ToUserResponse()
		}(user, index, &wg)
	}
	wg.Wait()

	log.Println("GetUsersWhoHaveSentMessageConcurrent - Time spent to run the task:", time.Since(start))

	return userResponses, nil
}

func (chr *ChatRepository) GetUsersWhoHaveSentMessageConcurrentWithRace() ([]*models.UserResponse, error) {
	var users []models.User
	wg := sync.WaitGroup{}
	start := time.Now()

	subQuery := chr.db.Raw("SELECT DISTINCT sender_id FROM messages")
	if err := chr.db.Raw("SELECT * FROM users WHERE id IN (?)", subQuery).Scan(&users).Error; err != nil {
		return nil, err
	}

	var userResponses = make([]*models.UserResponse, len(users))
	for _, user := range users {
		wg.Add(1)
		go func(user models.User, wg *sync.WaitGroup) {
			defer wg.Done()
			userResponses = append(userResponses, user.ToUserResponse())
		}(user, &wg)
	}
	wg.Wait()

	log.Println("GetUsersWhoHaveSentMessageConcurrent - Time spent to run the task:", time.Since(start))

	return userResponses, nil
}

func (chr *ChatRepository) GetUsersWhoHaveSentMessageConcurrentWithMutex() ([]*models.UserResponse, error) {
	var users []models.User
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	start := time.Now()

	subQuery := chr.db.Raw("SELECT DISTINCT sender_id FROM messages")
	if err := chr.db.Raw("SELECT * FROM users WHERE id IN (?)", subQuery).Scan(&users).Error; err != nil {
		return nil, err
	}

	var userResponses []*models.UserResponse
	for _, user := range users {
		wg.Add(1)
		go func(user models.User, wg *sync.WaitGroup, mu *sync.Mutex) {
			defer wg.Done()
			mu.Lock()
			userResponses = append(userResponses, user.ToUserResponse())
			mu.Unlock()
		}(user, &wg, &mu)
	}
	wg.Wait()

	log.Println("GetUsersWhoHaveSentMessageConcurrent - Time spent to run the task:", time.Since(start))

	return userResponses, nil
}

func (chr *ChatRepository) GetUsersWhoHaveSentMessage() ([]*models.UserResponse, error) {
	var users []models.User
	start := time.Now()

	subQuery := chr.db.Raw("SELECT DISTINCT sender_id FROM messages")
	if err := chr.db.Raw("SELECT * FROM users WHERE id IN (?)", subQuery).Scan(&users).Error; err != nil {
		return nil, err
	}

	var userResponses []*models.UserResponse
	for _, user := range users {
		userResponses = append(userResponses, user.ToUserResponse())
	}

	log.Println("GetUsersWhoHaveSentMessage - Time spent to run the task:", time.Since(start))

	return userResponses, nil
}
