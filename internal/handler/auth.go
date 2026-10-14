package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Temirlan2199/file-upload/internal/dto"
)

func (h *Handler) register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondBindError(c, err)
		return
	}

	// c.Request.Context() отменится, если клиент оборвёт соединение —
	// тогда и запрос в базу прервётся, не тратя ресурсы впустую.
	user, err := h.auth.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.NewUserResponse(user))
}

func (h *Handler) login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondBindError(c, err)
		return
	}

	token, expiresAt, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(expiresAt).Seconds()),
	})
}

func (h *Handler) me(c *gin.Context) {
	user, err := h.auth.GetUser(c.Request.Context(), userIDFromContext(c))
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewUserResponse(user))
}
