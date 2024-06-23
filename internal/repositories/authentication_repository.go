package repositories

import (
	"log"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	"socketChat/internal/utils"
	"time"

	"gorm.io/gorm"
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

func (ar *AuthenticationRepository) Login(login *models.LoginRequestBody) (*models.UserResponse, string, []error) {
	var errors []error
	var user *models.User = ar.CheckIfUserExists(login.Email)
	if user == nil {
		errors = append(errors, errs.ErrUserNotFound)
		return nil, "", errors
	}
	log.Printf("Password: %v Hash: %v", login.Password, user.PasswordHash)
	if err := utils.CompareHashAndPassword(user.PasswordHash, login.Password); err != nil {
		errors = append(errors, errs.ErrWrongPassword)
		return nil, "", errors
	}

	return user.ToUserResponse(), user.Email, nil
}

func (ar *AuthenticationRepository) GetAllUsersWithPagination(page, size int) (*models.GetUsersResponse, []error) {
	var users []models.User
	var userResponses []*models.UserResponse
	var errors []error
	var total int64

	transactionErr := ar.db.Transaction(func(tx *gorm.DB) error {
		result := tx.
			Scopes(utils.Paginate(page, size)).
			Select([]string{"ID", "first_name", "last_name", "profile_photo", "is_online", "last_seen"}).
			Find(&users).
			Where("deleted_at IS NULL")
		if err := result.Error; err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return errs.ErrThereIsNoUser
		}
		if err := ar.db.Model(&models.User{}).Where("deleted_at IS NULL").Count(&total).Error; err != nil {
			return err
		}

		return nil
	})

	if transactionErr != nil {
		errors = append(errors, transactionErr)
		return nil, errors
	}

	for _, user := range users {
		userResponses = append(userResponses, user.ToUserResponse())
	}

	usersResponse := &models.GetUsersResponse{
		Users: userResponses,
		Page:  page,
		Size:  size,
		Total: total,
	}

	return usersResponse, errors
}

func (ar *AuthenticationRepository) GetSingleUser(id int) (*models.UserResponse, []error) {
	var errors []error
	var user models.User
	result := ar.db.Where("id = ? AND deleted_at IS NULL", id).First(&user)
	if err := result.Error; err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	if result.RowsAffected == 0 {
		errors = append(errors, errs.ErrUserNotFound)
		return nil, errors
	}
	return user.ToUserResponse(), nil
}

func (ar *AuthenticationRepository) UpdateUserProfilePhoto(id uint, photo string) []error {
	var errors []error
	result := ar.db.Model(&models.User{}).Where("id = ?", id).Update("profile_photo", photo)
	if err := result.Error; err != nil {
		errors = append(errors, err)
		return errors
	}
	if result.RowsAffected == 0 {
		errors = append(errors, errs.ErrUserNotFound)
		return errors
	}
	return nil
}

func (ar *AuthenticationRepository) UpdateUser(updateUserReq *models.UpdateUserRequest) (*models.ProfileResponse, []error) {
	var errors []error
	var user models.User
	updates := map[string]interface{}{
		"first_name": updateUserReq.FirstName,
		"last_name":  updateUserReq.LastName,
	}
	result := ar.db.Model(&models.User{}).Where("id = ?", updateUserReq.ID).Updates(updates).First(&user, "id = ?", updateUserReq.ID)
	if err := result.Error; err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	if result.RowsAffected == 0 {
		errors = append(errors, errs.ErrUserNotFound)
		return nil, errors
	}

	return user.ToProfileResponse(), nil
}

func (ar *AuthenticationRepository) GetNotContactedYetUsers(userID uint, page, size int) (*models.GetUsersResponse, []error) {
	var errors []error
	var userResponses []*models.UserResponse
	var users []models.User

	// Find users that have a conversation with the logged in user
	subQuery := ar.db.Table("conversation_members").
		Select("user_id").
		Where("conversation_id IN (?)",
			ar.db.Table("conversation_members").
				Select("conversation_id").
				Where("user_id = ?", userID),
		)

	// Find users that have not been contacted yet
	result := ar.db.Table("users").
		Scopes(utils.Paginate(page, size)).
		Where("id NOT IN (?)", subQuery).
		Order("is_online DESC").
		Find(&users)

	if err := result.Error; err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	if len(users) == 0 {
		errors = append(errors, errs.ErrThereIsNoUser)
		return nil, errors
	}

	for _, user := range users {
		userResponses = append(userResponses, user.ToUserResponse())
	}

	usersResponse := &models.GetUsersResponse{
		Users: userResponses,
		Page:  page,
		Size:  size,
		Total: int64(len(users)),
	}

	return usersResponse, nil

}

func (ar *AuthenticationRepository) GetUserProfile(id int) (*models.ProfileResponse, []error) {
	var errors []error
	var user models.User
	result := ar.db.Where("id = ? AND deleted_at IS NULL", id).First(&user)
	if err := result.Error; err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	if result.RowsAffected == 0 {
		errors = append(errors, errs.ErrUserNotFound)
		return nil, errors
	}
	return user.ToProfileResponse(), nil
}

func (ar *AuthenticationRepository) SetUserOnlineStatus(userID uint, status bool) (bool, *time.Time, error) {
	lastSeen := time.Now()
	result := ar.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"is_online": status,
			"last_seen": lastSeen,
		})
	if err := result.Error; err != nil {
		return false, nil, err
	}
	if result.RowsAffected == 0 {
		return false, nil, errs.ErrUserNotFound
	}
	return status, &lastSeen, nil
}

func (ar *AuthenticationRepository) GetUserOnlineStatus(userID uint) (bool, *time.Time, error) {
	var user models.User
	result := ar.db.Select("is_online", "last_seen").
		Where("id = ?", userID).
		First(&user)

	if err := result.Error; err != nil {
		return false, nil, err
	}
	if result.RowsAffected == 0 {
		return false, nil, errs.ErrUserNotFound
	}
	return user.IsOnline, user.LastSeen, nil
}
