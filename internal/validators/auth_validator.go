package validators

import (
	"errors"
	"log"
	"regexp"
	"socketChat/internal/models"
)

func ValidateUser(user *models.User) error {
	if user == nil {
		return errors.New("user is nil")
	}

	if user.FirstName == "" || len(user.FirstName) < 2 {
		return errors.New("first name is empty or too short")
	}

	if user.LastName == "" || len(user.LastName) < 2 {
		return errors.New("last name is empty or too short")
	}

	if user.Email == "" {
		return errors.New("email is empty")
	}

	if !ValidatePassword(user.Password) {
		return errors.New("password is invalid")
	}

	return nil
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
		log.Println("Error compiling regular expression:", err)
		return false
	}

	// Match the password against the pattern
	return regex.MatchString(password)
}
