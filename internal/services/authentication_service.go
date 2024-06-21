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
	if len(validationErrs) > 0 {
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

	user, email, err := as.authRepo.Login(loginData)
	if err != nil {
		errors = append(errors, err...)
		return nil, errors
	}

	jwtExpiration := time.Now().Add(time.Duration(as.config.Viper.GetInt("jwt.expiration_time")) * time.Hour).Unix()
	token, jwtErr := utils.CreateJwtToken(
		user.ID,
		email,
		user.FirstName,
		user.LastName,
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
		errrors = append(errrors, errs.ErrInvalidPageOrSize)
		return nil, errrors
	}
	return as.authRepo.GetAllUsersWithPagination(page, size)
}

func (as *AuthenticationService) GetSingleUser(id int) (*models.UserResponse, []error) {
	var errrors []error

	if id <= 0 {
		errrors = append(errrors, errs.ErrInvalidParams)
		return nil, errrors
	}

	userResponse, getUserErrs := as.authRepo.GetSingleUser(id)

	if len(getUserErrs) > 0 {
		errrors = append(errrors, getUserErrs...)
		return nil, errrors
	}
	if userResponse == nil {
		errrors = append(errrors, errs.ErrUserNotFound)
		return nil, errrors
	}

	return userResponse, nil
}

func (as *AuthenticationService) UpdateUserProfilePhoto(id uint, photo string) []error {
	var errors []error
	if id <= 0 {
		errors = append(errors, errs.ErrInvalidParams)
		return errors
	}
	return as.authRepo.UpdateUserProfilePhoto(id, photo)
}

func (as *AuthenticationService) UpdateUser(user *models.UpdateUserRequest) (*models.ProfileResponse, []error) {
	var errors []error
	if user.ID <= 0 {
		errors = append(errors, errs.ErrInvalidParams)
		return nil, errors
	}

	if user.FirstName == "" || user.LastName == "" {
		errors = append(errors, errs.ErrInvalidParams)
		return nil, errors
	}
	return as.authRepo.UpdateUser(user)
}

func (as *AuthenticationService) GetNotContactedYetUsers(userID uint, page, size int) (*models.GetUsersResponse, []error) {
	var errors []error
	if page < 0 || size < 0 {
		errors = append(errors, errs.ErrInvalidPageOrSize)
		return nil, errors
	}
	return as.authRepo.GetNotContactedYetUsers(userID, page, size)
}

func (as *AuthenticationService) GetUserProfile(userID uint) (*models.ProfileResponse, []error) {
	var errors []error
	if userID <= 0 {
		errors = append(errors, errs.ErrInvalidParams)
		return nil, errors
	}
	return as.authRepo.GetUserProfile(int(userID))
}

func (as *AuthenticationService) SetUserOnlineStatus(userID uint, status bool) (bool, *time.Time, error) {
	return as.authRepo.SetUserOnlineStatus(userID, status)
}

func (as *AuthenticationService) GetUserOnlineStatus(userID uint) (bool, *time.Time, error) {
	return as.authRepo.GetUserOnlineStatus(userID)
}