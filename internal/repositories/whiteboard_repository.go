package repositories

import (
	"socketChat/internal/errs"
	"socketChat/internal/models"

	"gorm.io/gorm"
)

type WhiteboardRepository struct {
	db *gorm.DB
}

func NewWhiteboardRepository(db *gorm.DB) *WhiteboardRepository {
	return &WhiteboardRepository{
		db: db,
	}
}

func (wr *WhiteboardRepository) CreateNewWhiteboard(whiteboard *models.Whiteboard) (*models.Whiteboard, error) {
	result := wr.db.Create(whiteboard)
	if err := result.Error; err != nil {
		return nil, err
	}
	if result.RowsAffected <= 0 {
		return nil, errs.ErrWhiteboardCreationFailed
	}
	return whiteboard, nil
}

func (wr *WhiteboardRepository) FindConversationWhiteboard(conversationID uint) (*models.Whiteboard, error) {
	var whiteboard models.Whiteboard
	result := wr.db.Where("conversation_id = ?", conversationID).Last(&whiteboard)
	if err := result.Error; err != nil {
		return nil, err
	}
	if result.RowsAffected == 0 {
		return nil, errs.ErrNoWhiteboardFoundForThisConversation
	}
	return &whiteboard, nil
}