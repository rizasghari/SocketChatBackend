package repositories

import "gorm.io/gorm"

type AuthenticationRepository struct {
	db *gorm.DB
}

func NewAuthenticationRepository(db *gorm.DB) *AuthenticationRepository {
	return &AuthenticationRepository{
		db: db,
	}
}
