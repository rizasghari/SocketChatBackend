package services

import (
	"log"
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
		config:   config,
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
	password, err := utils.HashPassword(user.Password)
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

	jwtExpiration := time.Now().Add(time.Duration(as.config.Viper.GetInt("jwt.expiration_time")) * time.Second).Unix()
	token, jwtErr := utils.CreateJwtToken(
		user.ID,
		user.Email,
		user.FirstName,
		user.LastName,
		utils.GetJwtKey(),
		time.Unix(jwtExpiration, 0),
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

func (as *AuthenticationService) GetAllUsersWithPagination(page, size int) (*models.GetUsersResponse, []error) {
	var errrors []error
	if page < 0 || size < 0 {
		log.Println("Page or size < 0")
		errrors = append(errrors, errs.ErrInvalidPageOrSize)
		return nil, errrors
	}
	offset := (page - 1) * size
	log.Println("Offset: ", offset)
	return as.authRepo.GetAllUsersWithPagination(page, size, offset)
}