package repositories

import (
	"gorm.io/gorm"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	"socketChat/internal/utils"
)

type AuthenticationRepository struct {
	db *gorm.DB
}

func NewAuthenticationRepository(db *gorm.DB) *AuthenticationRepository {
	return &AuthenticationRepository{
		db: db,
	}
}

func (ar *AuthenticationRepository) CreateUser(user *models.User) (*models.User, []error) {
	var errors []error
	result := ar.db.Create(user)
	if result.Error != nil {
		errors = append(errors, result.Error)
		return nil, errors
	}
	if result.RowsAffected == 0 {
		errors = append(errors, errs.ErrUserNotFound)
		return nil, errors
	}
	return user, nil
}

func (ar *AuthenticationRepository) CheckIfUserExists(email string) *models.User {
	var user models.User
	result := ar.db.Where("email = ?", email).First(&user)
	if result.Error == nil && result.RowsAffected > 0 {
		return &user
	}
	return nil
}

func (ar *AuthenticationRepository) Login(login *models.LoginRequestBody) (*models.User, []error) {
	var errors []error
	var user *models.User
	user = ar.CheckIfUserExists(login.Email)
	if user == nil {
		errors := append(errors, errs.ErrUserNotFound)
		return nil, errors
	}
	if err := utils.CompareHashAndPassword(user.PasswordHash, login.Password); err != nil {
		errors := append(errors, errs.ErrWrongPassword)
		return nil, errors
	}
	return user, nil
}
