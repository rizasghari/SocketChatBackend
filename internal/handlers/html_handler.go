package handlers

import (
	"net/http"
	"socketChat/internal/services"
	"socketChat/internal/web"

	"github.com/gin-gonic/gin"
)

type HtmlHandler struct {
	authService        *services.AuthenticationService
	chatService        *services.ChatService
	fileManagerService *services.FileManagerService
}

func NewHtmlHandler(
	authService *services.AuthenticationService,
	chatService *services.ChatService,
	fileManagerService *services.FileManagerService,
) *HtmlHandler {
	return &HtmlHandler{
		authService:        authService,
		chatService:        chatService,
		fileManagerService: fileManagerService,
	}
}

func (hh *HtmlHandler) NotFound(ctx *gin.Context) {
	ctx.HTML(http.StatusNotFound, "", web.NotFound())
}

func (hh *HtmlHandler) Index(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "", web.Index(false, "home"))
}
