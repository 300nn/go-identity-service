package auth

import (
	"CrudTutorialProject/internal/ratelimit"
	"CrudTutorialProject/internal/response"
	"CrudTutorialProject/internal/validation"
	"log/slog"
	"net"
	"net/http"
	"strings"
)

type Handler struct {
	service         *Service
	logger          *slog.Logger
	validator       *validation.Validator
	limiter         RateLimiter
	rateLimitConfig RateLimitConfig
}

func NewHandler(
	service *Service,
	logger *slog.Logger,
	validator *validation.Validator,
	limiter *ratelimit.Limiter,
	rateLimitConfig RateLimitConfig,
) *Handler {
	return &Handler{
		service:         service,
		logger:          logger,
		validator:       validator,
		limiter:         limiter,
		rateLimitConfig: rateLimitConfig,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware *MiddleWare) {
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /auth/refresh", h.Refresh)
	mux.HandleFunc("POST /auth/logout", h.Logout)

	mux.Handle("GET /auth/me", authMiddleware.RequireAuth(http.HandlerFunc(h.Me)))
}

func (h *Handler) rejectRateLimited(w http.ResponseWriter) {
	response.SendError(
		w,
		http.StatusTooManyRequests,
		"rate_limited",
		"Too many requests",
	)
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

	if h.limiter != nil {
		key := "auth:register:ip:" + clientIP(r)
		if !h.limiter.Allow(key, h.rateLimitConfig.RegisterLimit, h.rateLimitConfig.RegisterWindow) {
			h.rejectRateLimited(w)
			return
		}
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

	if h.limiter != nil {
		email := strings.TrimSpace(strings.ToLower(req.Email))
		ip := clientIP(r)
		key := "auth:login:ip:" + email + ":" + ip

		if !h.limiter.Allow(key, h.rateLimitConfig.LoginLimit, h.rateLimitConfig.LoginWindow) {
			h.rejectRateLimited(w)
			return
		}
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

	if h.limiter != nil {
		key := "auth:refresh:ip:" + clientIP(r)

		if !h.limiter.Allow(
			key,
			h.rateLimitConfig.RefreshLimit,
			h.rateLimitConfig.RefreshWindow,
		) {
			h.rejectRateLimited(w)
			return
		}
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

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}

	return "unknown"
}
