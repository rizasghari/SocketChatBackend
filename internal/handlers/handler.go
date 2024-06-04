package handlers

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"socketChat/internal/models"
	"socketChat/internal/services"
)

var jwtKey = []byte("aycEW3OtV+axBFZQL4cplAVRFMhSEc+xRrcHXxhTM8U=")

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

func (h *Handler) Login(ctx *gin.Context) {
	var login *models.Login
	err := ctx.BindJSON(login)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	}
	user, err := h.authService.Login(login)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	}

	ctx.JSON(http.StatusOK, user)
}

func (h *Handler) Register(ctx *gin.Context) {
	var user models.User
	err := ctx.BindJSON(&user)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	}
	log.Println("Register - user: ", user)
	register, err := h.authService.Register(&user)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	}

	ctx.JSON(http.StatusOK, register)
}
