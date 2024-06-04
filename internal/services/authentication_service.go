package services

import (
	"socketChat/internal/models"
	"socketChat/internal/repositories"
	"socketChat/internal/utils"
	"socketChat/internal/validators"
)

type AuthenticationService struct {
	authRepo *repositories.AuthenticationRepository
}

func NewAuthenticationService(authRepo *repositories.AuthenticationRepository) *AuthenticationService {
	return &AuthenticationService{
		authRepo: authRepo,
	}
}

func (as *AuthenticationService) Register(user *models.User) (*models.User, error) {
	err := validators.ValidateUser(user)
	if err != nil {
		return nil, err
	}
	user.PasswordHash, err = utils.HashPassword(user.PasswordHash)
	return as.authRepo.CreateUser(user)
}

func (as *AuthenticationService) Login(login *models.Login) (*models.User, error) {
	return nil, nil
}
