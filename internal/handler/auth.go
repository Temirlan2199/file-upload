package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Temirlan2199/file-upload/internal/dto"
	"github.com/Temirlan2199/file-upload/internal/service"
)

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// r.Context() отменится, если клиент оборвёт соединение.
	user, err := h.auth.Register(r.Context(), req.Email, req.Password)
	if errors.Is(err, service.ErrEmailTaken) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		h.internalError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dto.NewUserResponse(user))
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if errors.Is(err, service.ErrInvalidCredentials) {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err != nil {
		h.internalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.TokenResponse{Token: token})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, err := h.auth.GetUser(r.Context(), userIDFromContext(r.Context()))
	if err != nil {
		h.internalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.NewUserResponse(user))
}
