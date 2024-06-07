package handlers

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"socketChat/internal/enums"
	"socketChat/internal/errs"
	"socketChat/internal/models"
	"socketChat/internal/msgs"
	"socketChat/internal/services"
	"socketChat/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	authService        *services.AuthenticationService
	chatService        *services.ChatService
	fileManagerService *services.FileManagerService
}

func NewHandler(
	authService *services.AuthenticationService,
	chatService *services.ChatService,
	fileManagerService *services.FileManagerService,
) *Handler {
	return &Handler{
		authService:        authService,
		chatService:        chatService,
		fileManagerService: fileManagerService,
	}
}

func (h *Handler) Index(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "index.html", nil)
}

func (h *Handler) Login(ctx *gin.Context) {
	var errors []error

	var loginData models.LoginRequestBody
	err := ctx.BindJSON(&loginData)
	if err != nil {
		log.Println("Error login data json binding:", err)
		errors = append(errors, errs.ErrInvalidRequestBody)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  errors,
		})
		return
	}

	loginResponse, loginErrs := h.authService.Login(&loginData)
	if loginErrs != nil && len(loginErrs) > 0 {
		errors = append(errors, loginErrs...)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  errors,
		})
		return
	}

	ctx.JSON(http.StatusOK, models.Response{
		Success: true,
		Message: msgs.MsgOperationSuccessful,
		Data:    loginResponse,
	})
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

	_, registerErrs := h.authService.Register(&user)
	if len(registerErrs) > 0 {
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
	})
}

func (h *Handler) CreateConversation(ctx *gin.Context) {
	var errors []error

	var createConversationRequestBody models.CreateConversationRequestBody
	err := ctx.BindJSON(&createConversationRequestBody)
	if err != nil {
		errors = append(errors, errs.ErrInvalidRequestBody)
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  errors,
		})
		return
	}

	conversation, errors := h.chatService.CreateConversation(&createConversationRequestBody)
	if len(errors) > 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  errors,
		})
		return
	}

	ctx.JSON(http.StatusOK, models.Response{
		Success: true,
		Message: msgs.MsgOperationSuccessful,
		Data:    conversation,
	})
}

func (h *Handler) GetAllUsersWithPagination(ctx *gin.Context) {
	page := ctx.Query("page")
	size := ctx.Query("size")

	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		pageInt = 1
	}

	sizeInt, err := strconv.Atoi(size)
	if err != nil || sizeInt < 1 {
		sizeInt = 10
	}

	response, errs := h.authService.GetAllUsersWithPagination(pageInt, sizeInt)
	if len(errs) > 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  errs,
		})
		return
	}
	ctx.JSON(http.StatusOK, models.Response{
		Success: true,
		Message: msgs.MsgOperationSuccessful,
		Data:    response,
	})
}

func (h *Handler) GetSingleUser(ctx *gin.Context) {
	id := ctx.Param("id")

	idInt, err := strconv.Atoi(id)
	if err != nil || idInt < 1 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrInvalidParams},
		})
		return
	}

	user, errs := h.authService.GetSingleUser(idInt)
	if len(errs) > 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  errs,
		})
		return
	}
	ctx.JSON(http.StatusOK, models.Response{
		Success: true,
		Message: msgs.MsgOperationSuccessful,
		Data:    user,
	})
}

func (h *Handler) GetUserConversations(ctx *gin.Context) {
	id := ctx.Param("id")
	page := ctx.Query("page")
	size := ctx.Query("size")

	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		pageInt = 1
	}

	sizeInt, err := strconv.Atoi(size)
	if err != nil || sizeInt < 1 {
		sizeInt = 10
	}

	idInt, err := strconv.Atoi(id)
	if err != nil || idInt < 1 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrInvalidParams},
		})
		return
	}

	conversationsResponse, errs := h.chatService.GetUserConversations(uint(idInt), pageInt, sizeInt)
	if len(errs) > 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  errs,
		})
		return
	}
	ctx.JSON(http.StatusOK, models.Response{
		Success: true,
		Message: msgs.MsgOperationSuccessful,
		Data:    conversationsResponse,
	})
}

func (h *Handler) GetUserConversationsByToken(ctx *gin.Context) {
	page := ctx.Query("page")
	size := ctx.Query("size")

	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		pageInt = 1
	}

	sizeInt, err := strconv.Atoi(size)
	if err != nil || sizeInt < 1 {
		sizeInt = 10
	}

	idInt := ctx.MustGet("user_id").(uint)
	if err != nil || idInt < 1 {
		log.Println("User id not found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrUnauthorized},
		})
		return
	}

	conversationsResponse, errs := h.chatService.GetUserConversations(idInt, pageInt, sizeInt)
	if len(errs) > 0 {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  errs,
		})
		return
	}
	ctx.JSON(http.StatusOK, models.Response{
		Success: true,
		Message: msgs.MsgOperationSuccessful,
		Data:    conversationsResponse,
	})
}

func (h *Handler) UploadUserProfilePhoto(ctx *gin.Context) {
	userID := utils.GetUserIdFromContext(ctx)
	if userID < 1 {
		log.Println("User id not found")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrUnauthorized},
		})
		return
	}

	file, err := ctx.FormFile("profile_photo")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrNoFileUploaded},
		})
		return
	}

	src, err := file.Open()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrUnableToOpenUploadedFile},
		})
		return
	}
	defer src.Close()

	// Generate a unique file name based on user ID and original file extension
	fileExt := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("user_profile_photo_%s%s", strconv.Itoa(int(userID)), fileExt)

	// Upload the file to MinIO
	url, err := h.fileManagerService.UploadUserProfilePhoto(fileName, src, file.Size, file.Header.Get("Content-Type"), enums.FILE_BUCKET_USER_PROFILE)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrUnableToUploadFile},
		})
		return
	}

	// Update the user profile photo URL in the database
	if updateErrs := h.authService.UpdateUserProfilePhoto(userID, url); updateErrs != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, models.Response{
			Success: false,
			Message: msgs.MsgOperationFailed,
			Errors:  []error{errs.ErrUnableToUpdateProfilePhoto},
		})
		return
	}

	ctx.JSON(http.StatusOK, models.Response{
		Success: true,
		Message: msgs.MsgOperationSuccessful,
		Data:    url,
	})
}
