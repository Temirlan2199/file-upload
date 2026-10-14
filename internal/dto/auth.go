// Package dto — структуры запросов и ответов API.
// Они отделены от model: например, PasswordHash есть в model.User,
// но в JSON-ответ он попасть не должен.
package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/Temirlan2199/file-upload/internal/model"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r RegisterRequest) Validate() error {
	if !strings.Contains(r.Email, "@") {
		return errors.New("invalid email")
	}
	if len(r.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUserResponse(u *model.User) UserResponse {
	return UserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}
