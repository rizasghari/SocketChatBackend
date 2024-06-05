package handlers

import (
	"net/http"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	"socketChat/internal/msgs"
	"socketChat/internal/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handler) MustAuthenticateMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var jwtToken string
		jwtTokenFromHeader := ctx.GetHeader("Authorization")
		if jwtTokenFromHeader != "" {
			if strings.Contains(jwtTokenFromHeader, "Bearer") {
				jwtTokenFromHeader = strings.Replace(jwtTokenFromHeader, "Bearer ", "", 1)
			}
		}

		if jwtTokenFromHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
				Success: false,
				Message: msgs.MsgOperationFailed,
				Errors:  []error{errs.ErrUnauthorized},
			})
			return
		}

		claims, err := utils.VerifyToken(jwtToken, utils.GetJwtKey())
		if err != nil {
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
