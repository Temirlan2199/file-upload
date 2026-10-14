// Package dto — структуры запросов и ответов API (Data Transfer Objects).
// Они описывают контракт API и отделены от model: например, PasswordHash
// есть в model.User, но никогда не должен попасть в JSON-ответ.
package dto

import (
	"time"

	"github.com/Temirlan2199/file-upload/internal/model"
)

// Теги binding проверяет gin (через go-playground/validator) в ShouldBindJSON.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"` // секунд до истечения
}

// NewUserResponse переводит доменную модель в ответ API.
func NewUserResponse(u *model.User) UserResponse {
	return UserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}
