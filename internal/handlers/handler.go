package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	"socketChat/internal/msgs"
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
	var errors []error

	var user models.User
	err := ctx.BindJSON(&user)
	if err != nil {
		errors = append(errors, errs.ErrInvalidRequestBody)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  errors,
		})
		return
	}

	register, registerErrs := h.authService.Register(&user)
	if registerErrs != nil && len(registerErrs) > 0 {
		errors = append(errors, registerErrs...)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  errors,
		})
		return
	}

	ctx.JSON(http.StatusOK, models.Response{
		Success: true,
		Message: msgs.MsgUserCreatedSuccessfully,
		Data:    register,
	})
}
