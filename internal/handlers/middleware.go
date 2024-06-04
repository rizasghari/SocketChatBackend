package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"socketChat/internal/utils"
	"strings"
)

func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var jwtToken string
		jwtTokenFromHeader := ctx.GetHeader("Authorization")
		if jwtTokenFromHeader != "" {
			if strings.Contains(jwtTokenFromHeader, "Bearer") {
				jwtToken = strings.Replace(jwtTokenFromHeader, "Bearer ", "", 1)
			} else {
				jwtToken = jwtTokenFromHeader
			}
		} else {
			jwtTokenFromCookie, err := ctx.Cookie("jwt_token")
			if err != nil {
				ctx.Redirect(http.StatusFound, "/login")
				return
			}
			jwtToken = jwtTokenFromCookie
		}

		if jwtToken == "" {
			ctx.Redirect(http.StatusFound, "/login")
			return
		}

		claims, err := utils.VerifyToken(jwtToken, utils.GetJwtKey())
		if err != nil {
			ctx.Redirect(http.StatusFound, "/login")
			return
		}

		ctx.Set("user_id", claims.ID)
		ctx.Set("user_email", claims.Email)
		ctx.Set("authenticated", true)
		ctx.Next()
	}
}
