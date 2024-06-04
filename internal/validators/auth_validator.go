package validators

import (
	"log"
	"regexp"
	"socketChat/internal/errs"
	"socketChat/internal/models"
)

func ValidateUser(user *models.User) []error {
	var errors []error
	if user == nil {
		errors = append(errors, errs.ErrInvalidUser)
		return errors
	}

	if user.Email == "" || !ValidateEmail(user.Email) {
		errors = append(errors, errs.ErrInvalidEmail)
	}

	if passwordValidationErrs := ValidatePassword(user.Password); passwordValidationErrs != nil &&
		len(passwordValidationErrs) > 0 {
		errors = append(errors, passwordValidationErrs...)
	}

	if user.FirstName == "" || len(user.FirstName) < 2 {
		errors = append(errors, errs.ErrFirstName)
	}

	if user.LastName == "" || len(user.LastName) < 2 {
		errors = append(errors, errs.ErrLastName)
	}
	return errors
}

func ValidateEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	regex, err := regexp.Compile(pattern)
	if err != nil {
		log.Println("Error compiling regular expression:", err)
		return false
	}
	return regex.MatchString(email)
}

func ValidatePassword(password string) []error {
	var errors []error

	// Check the length and valid characters
	if len(password) < 8 {
		errors = append(errors, errs.ErrPasswordAtLeast8Characters)
	}

	// Check for at least one digit
	hasDigit := `[0-9]`
	if matched, _ := regexp.MatchString(hasDigit, password); !matched {
		errors = append(errors, errs.ErrPasswordAtLeastOneDigit)
	}

	// Check for at least one lowercase letter
	hasLower := `[a-z]`
	if matched, _ := regexp.MatchString(hasLower, password); !matched {
		errors = append(errors, errs.ErrPasswordAtLeastOneLower)
	}

	// Check for at least one uppercase letter
	hasUpper := `[A-Z]`
	if matched, _ := regexp.MatchString(hasUpper, password); !matched {
		errors = append(errors, errs.ErrPasswordAtLeastOneUpper)
	}

	// Check for at least one special character
	hasSpecial := `[@#$%^&+=!]`
	if matched, _ := regexp.MatchString(hasSpecial, password); !matched {
		errors = append(errors, errs.ErrPasswordAtLeastOneSpecial)
	}

	return errors
}
