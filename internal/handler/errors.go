package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/Temirlan2199/file-upload/internal/dto"
	"github.com/Temirlan2199/file-upload/internal/service"
)

// respondError переводит ошибку сервиса в HTTP-ответ. Это единственное место,
// где решается, какой статус получит клиент.
func (h *Handler) respondError(c *gin.Context, err error) {
	status, msg := http.StatusInternalServerError, "internal server error"

	switch {
	case errors.Is(err, service.ErrEmailTaken):
		status, msg = http.StatusConflict, err.Error()
	case errors.Is(err, service.ErrInvalidCredentials):
		status, msg = http.StatusUnauthorized, err.Error()
	case errors.Is(err, service.ErrPasswordTooLong):
		status, msg = http.StatusBadRequest, err.Error()
	case errors.Is(err, service.ErrUserNotFound):
		status, msg = http.StatusNotFound, err.Error()
	default:
		// Неизвестная ошибка: подробности — в лог, клиенту — общий текст.
		// Внутренние детали (SQL, стектрейсы) наружу не отдаём.
		h.log.Error("request failed",
			slog.String("path", c.Request.URL.Path),
			slog.Any("error", err),
		)
	}

	c.AbortWithStatusJSON(status, dto.ErrorResponse{Error: msg})
}

// respondBindError отвечает 400 на невалидное тело запроса.
// Ошибки валидатора превращаем в понятный map поле → причина.
func (h *Handler) respondBindError(c *gin.Context, err error) {
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		// Например, тело — не JSON.
		c.AbortWithStatusJSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body"})
		return
	}

	details := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		details[fe.Field()] = validationMessage(fe)
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, dto.ErrorResponse{
		Error:   "validation failed",
		Details: details,
	})
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "min":
		return "must be at least " + fe.Param() + " characters"
	case "max":
		return "must be at most " + fe.Param() + " characters"
	default:
		return "is invalid"
	}
}

var registerOnce sync.Once

// registerValidatorJSONNames заставляет валидатор называть поля так же,
// как в JSON ("email"), а не как в Go-структуре ("Email").
func registerValidatorJSONNames() {
	registerOnce.Do(func() {
		v, ok := binding.Validator.Engine().(*validator.Validate)
		if !ok {
			return
		}
		v.RegisterTagNameFunc(func(f reflect.StructField) string {
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "-" {
				return ""
			}
			return name
		})
	})
}
