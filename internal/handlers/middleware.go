package handlers

import (
	"log"
	"net/http"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	"socketChat/internal/msgs"
	"socketChat/internal/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func MustAuthenticateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		jwtTokenFromHeader := ctx.GetHeader("Authorization")
		if jwtTokenFromHeader != "" {
			if strings.Contains(jwtTokenFromHeader, "Bearer") {
				jwtTokenFromHeader = strings.Replace(jwtTokenFromHeader, "Bearer ", "", 1)
			}
		}

		if jwtTokenFromHeader == "" {
			log.Println("JWT token not found")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
				Success: false,
				Message: msgs.MsgOperationFailed,
				Errors:  []error{errs.ErrUnauthorized},
			})
			return
		}

		claims, err := utils.VerifyToken(jwtTokenFromHeader)
		if err != nil {
			log.Println("JWT token verification failed:", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
				Success: false,
				Message: msgs.MsgYouMustLoginFirst,
				Errors:  []error{errs.ErrUnauthorized},
			})
			return
		}

		ctx.Set("user_id", claims.ID)
		ctx.Set("user_email", claims.Email)
		ctx.Set("user_first_name", claims.FirstName)
		ctx.Set("user_last_name", claims.LastName)
		ctx.Set("authenticated", true)
		ctx.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
        ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

        if ctx.Request.Method == "OPTIONS" {
            ctx.AbortWithStatus(http.StatusNoContent)
            return
        }

        ctx.Next()
    }
}