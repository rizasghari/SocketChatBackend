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

func (wr *WhiteboardRepository) CreateNewWhiteboard(whiteboard *models.Whiteboard) error {
	result := wr.db.Create(whiteboard)
	if err := result.Error; err != nil {
		return err
	}
	if result.RowsAffected <= 0 {
		return errs.ErrWhiteboardCreationFailed
	}
	return nil
}

