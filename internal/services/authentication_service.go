package services

import "socketChat/internal/repositories"

type AuthenticationService struct {
	authRepo *repositories.AuthenticationRepository
}

func NewAuthenticationService(authRepo *repositories.AuthenticationRepository) *AuthenticationService {
	return &AuthenticationService{
		authRepo: authRepo,
	}
}
