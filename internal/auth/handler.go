package auth

import (
	"CrudTutorialProject/internal/response"
	"CrudTutorialProject/internal/validation"
	"log/slog"
	"net/http"
)

type Handler struct {
	service   *Service
	logger    *slog.Logger
	validator *validation.Validator
}

func NewHandler(service *Service, logger *slog.Logger, validator *validation.Validator) *Handler {
	return &Handler{
		service:   service,
		logger:    logger,
		validator: validator,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware *MiddleWare) {
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /auth/refresh", h.Refresh)
	mux.HandleFunc("POST /auth/logout", h.Logout)

	mux.Handle("GET /auth/me", authMiddleware.RequireAuth(http.HandlerFunc(h.Me)))
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	req, ok := response.DecodeJSON[RegisterRequest](w, r)
	if !ok {
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	result, err := h.service.Register(r.Context(), req)
	if err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	req, ok := response.DecodeJSON[LoginRequest](w, r)
	if !ok {
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	result, err := h.service.Login(r.Context(), req)

	if err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	req, ok := response.DecodeJSON[RefreshRequest](w, r)
	if !ok {
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	result, err := h.service.Refresh(r.Context(), req)
	if err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	req, ok := response.DecodeJSON[LogoutRequest](w, r)

	if !ok {
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	if err := h.service.Logout(r.Context(), req); err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())

	if !ok {
		response.HandleError(w, h.logger, NewUnauthorizedError())
		return
	}

	result, err := h.service.Me(r.Context(), principal.UserID)

	if err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
