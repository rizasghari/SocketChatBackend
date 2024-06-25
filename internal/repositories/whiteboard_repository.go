package repositories

import (
	"socketChat/internal/models/whiteboard"

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

func (wr *WhiteboardRepository) CreateNewWhiteboard(whiteboard *whiteboard.Whiteboard) (*whiteboard.Whiteboard, []error) {
	return nil, nil
}
