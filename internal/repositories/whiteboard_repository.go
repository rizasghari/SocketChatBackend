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
	txErr := wr.db.Transaction(func(tx *gorm.DB) error {
		// Create the whiteboard
		whiteboardCreationResult := tx.Create(whiteboard)
		if err := whiteboardCreationResult.Error; err != nil {
			return err
		}
		if whiteboardCreationResult.RowsAffected <= 0 {
			return errs.ErrWhiteboardCreationFailed
		}

		// Create drawns per each conversation member for the newly created whitboard
		var members []models.ConversationMember
		if err := tx.Where("conversation_id = ?", whiteboard.ConversationID).
			Find(&members).Error; err != nil {
			return err
		}

		for _, member := range members {
			drawn := models.Drawn{
				Drawer:       member.UserID,
				WhiteboardID: whiteboard.ID,
			}
			if err := tx.Create(&drawn).Error; err != nil {
				return err
			}
			whiteboard.Drawns = append(whiteboard.Drawns, drawn)
		}

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	return whiteboard, nil
}

func (wr *WhiteboardRepository) FindConversationWhiteboard(conversationID uint) (*models.Whiteboard, error) {
	var whiteboard models.Whiteboard
	result := wr.db.
		Preload("Drawns").
		Where("conversation_id = ?", conversationID).
		Last(&whiteboard)
	if err := result.Error; err != nil {
		return nil, err
	}
	if result.RowsAffected == 0 {
		return nil, errs.ErrNoWhiteboardFoundForThisConversation
	}
	return &whiteboard, nil
}

func (wr *WhiteboardRepository) CreateNewDrawn(drawn *models.Drawn) (*models.Drawn, error) {
	result := wr.db.Create(drawn)
	if err := result.Error; err != nil {
		return nil, err
	}
	if result.RowsAffected <= 0 {
		return nil, errs.ErrWhiteboardCreationFailed
	}
	return drawn, nil
}

func (wr *WhiteboardRepository) FindWhiteboardDrawn(whiteboardID, drawer uint) (*models.Drawn, error) {
	var drawn models.Drawn
	result := wr.db.Where("whiteboard_id = ? AND user_id = ?", whiteboardID, drawer).Last(&drawn)
	if err := result.Error; err != nil {
		return nil, err
	}
	if result.RowsAffected == 0 {
		return nil, errs.ErrNoWhiteboardFoundForThisConversation
	}
	return &drawn, nil
}
