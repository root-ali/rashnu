package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/root-ali/rashnu/internal/identity"
	"go.uber.org/zap"
)

type contextKey string

const (
	ContextKeyRole  contextKey = "role"
	ContextKeyEmail contextKey = "email"
)

func (h *Handler) GetAllUserHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("About to get all users")
	users, err := h.identity.GetAll(r.Context())
	if err != nil {
		h.logger.Error("get all users", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
	return
}

func (h *Handler) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req identity.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.identity.Create(r.Context(), req); err != nil {
		if errors.Is(err, identity.ErrUserAlreadyExist) {
			writeError(w, http.StatusConflict, err)
			return
		}
		h.logger.Error("create user", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req identity.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.identity.Update(r.Context(), req); err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		h.logger.Error("update", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) UpdateUserPasswordHandler(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(ContextKeyEmail).(string)
	if !ok {
		h.logger.Error("update password error: email not found in context")
		writeError(w, http.StatusInternalServerError, errors.New("email not found in context"))
		return
	}

	var req identity.UpdateUserPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.Email = email

	if err := h.identity.UpdatePassword(r.Context(), req); err != nil {
		if errors.Is(err, identity.ErrUserNotFound) || errors.Is(err, identity.ErrPasswordMismatch) {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		h.logger.Error("update password error", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	h.logger.Debug("update password success", zap.Any("email", email))
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	email, ok := r.Context().Value(ContextKeyEmail).(string)
	if !ok {
		h.logger.Error("delete password error: email not found in context")
		writeError(w, http.StatusInternalServerError, errors.New("email not found in context"))
		return
	}
	h.logger.Debug("delete user", zap.Any("email requested", email), zap.Any("id", id))

	err := h.identity.Delete(r.Context(), id)
	if err != nil {
		h.logger.Error("delete user error", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	h.logger.Debug("delete user successfully", zap.Any("email", email), zap.Any("id", id))
	writeJSON(w, http.StatusNoContent, nil)
	return
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(ContextKeyEmail).(string)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("email not found in context"))
		return
	}
	if err := h.identity.Logout(r.Context(), email); err != nil {
		h.logger.Error("logout", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	h.logger.Info("user logged out", zap.String("email", email))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req identity.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	loginResponse, err := h.identity.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) || errors.Is(err, identity.ErrPasswordMismatch) {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		h.logger.Error("login", zap.Error(err))
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, loginResponse)
}
