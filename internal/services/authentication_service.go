package services

import (
	"socketChat/configs"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	"socketChat/internal/repositories"
	"socketChat/internal/utils"
	"socketChat/internal/validators"
	"time"
)

type AuthenticationService struct {
	authRepo *repositories.AuthenticationRepository
	config   *configs.Config
}

func NewAuthenticationService(
	authRepo *repositories.AuthenticationRepository,
	config *configs.Config,
) *AuthenticationService {
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

func (as *AuthenticationService) Login(loginData *models.LoginRequestBody) (*models.LoginResponse, []error) {
	var errors []error

	user, err := as.authRepo.Login(loginData)
	if err != nil {
		errors = append(errors, err...)
		return nil, errors
	}

	jwtExpiration := time.Now().Add(time.Duration(as.config.Viper.GetInt("jwt.expiration_time")) * time.Second)
	token, jwtErr := utils.CreateJwtToken(
		user.ID,
		user.Email,
		[]byte(utils.GenerateSecretKey()),
		jwtExpiration,
	)
	if jwtErr != nil {
		errors = append(errors, jwtErr)
		return nil, errors
	}

	return &models.LoginResponse{
		User:  *user,
		Token: token,
	}, nil
}

func (as *AuthenticationService) CheckIfUserExists(email string) bool {
	return as.authRepo.CheckIfUserExists(email) != nil
}
