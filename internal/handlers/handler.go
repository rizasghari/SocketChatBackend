package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"socketChat/internal/services"
)

type Handler struct {
	authService *services.AuthenticationService
}

func NewHandler(authService *services.AuthenticationService) *Handler {
	return &Handler{
		authService: authService,
	}
}

func (h *Handler) Index(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "index.html", nil)
}
