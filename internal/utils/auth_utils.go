package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"socketChat/internal/models"
	"time"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func CompareHashAndPassword(hashedPassword string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func GenerateSecretKey() string {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(key)
}

func CreateJwtToken(id uint, email, firstName, lastName string, secretKey []byte, expiration time.Time) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		models.Claims{
			ID:        id,
			Email:     email,
			FirstName: firstName,
			LastName:  lastName,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expiration),
			},
		})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func VerifyToken(tokenString string, secretKey []byte) (*models.Claims, error) {
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error parsing token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func IsAuthenticated(ctx *gin.Context, jwtKey []byte) bool {
	jwtToken, err := ctx.Cookie("jwt_token")
	if err != nil {
		return false
	}
	if _, err = VerifyToken(jwtToken, jwtKey); err != nil {
		return false
	}

	return true
}

func GetJwtKey() []byte {
	return []byte("aycEW3OtV+axBFZQL4cplAVRFMhSEc+xRrcHXxhTM8U=")
}
