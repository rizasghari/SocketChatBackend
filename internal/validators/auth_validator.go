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

	if !ValidatePassword(user.Password) {
		errors = append(errors, errs.ErrInvalidPassword)
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

func ValidatePassword(password string) bool {
	// ^: Matches the start of the string.
	// (?:[0-9a-zA-Z@#$%^&+=!]{8,}): Matches a string of at least 8 characters consisting of digits,
	// letters (uppercase and lowercase), and special characters (@#$%^&+=!).
	// (?:(.*[0-9])?(.*[a-z])?(.*[A-Z])?(.*[@#$%^&+=!])?): This part uses non-capturing groups ((?:...))
	// and optional groups (?(...)?) to ensure the presence of at least one digit, one lowercase letter,
	// one uppercase letter, and one special character.
	// $: Matches the end of the string.
	pattern := `^(?:[0-9a-zA-Z@#$%^&+=!]{8,})(?:(.*[0-9])?(.*[a-z])?(.*[A-Z])?(.*[@#$%^&+=!])?)$`

	// Compile the regular expression
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}

	// Match the password against the pattern
	return regex.MatchString(password)
}
