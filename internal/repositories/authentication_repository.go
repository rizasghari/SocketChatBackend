package repositories

import (
	"fmt"
	"gorm.io/gorm"
	"socketChat/internal/models"
)

type AuthenticationRepository struct {
	db *gorm.DB
}

func NewAuthenticationRepository(db *gorm.DB) *AuthenticationRepository {
	return &AuthenticationRepository{
		db: db,
	}
}

func (ar *AuthenticationRepository) CreateUser(user *models.User) (*models.User, error) {
	result := ar.db.Create(user)
	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("user not created")
	}

	return user, nil
}
