package services

import (
	"socketChat/internal/errs"
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

func (as *AuthenticationService) Register(user *models.User) (*models.User, []error) {
	var errors []error
	if as.CheckIfUserExists(user.Email) {
		errors = append(errors, errs.ErrUserAlreadyExists)
		return nil, errors
	}
	validationErrs := validators.ValidateUser(user)
	if validationErrs != nil && len(validationErrs) > 0 {
		errors = append(errors, validationErrs...)
		return nil, errors
	}
	password, err := utils.HashPassword(user.PasswordHash)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	user.PasswordHash = password
	return as.authRepo.CreateUser(user)
}

func (as *AuthenticationService) Login(login *models.Login) (*models.User, error) {
	return nil, nil
}

func (as *AuthenticationService) CheckIfUserExists(email string) bool {
	return as.authRepo.CheckIfUserExists(email) != nil
}
