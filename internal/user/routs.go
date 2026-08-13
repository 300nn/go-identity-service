package user

import (
	"net/http"
)

type MiddlewareFunc func(http.Handler) http.Handler

func (h *Handler) RegisterRouts(
	mux *http.ServeMux,
	requireAdmin MiddlewareFunc,
	requireSelfOrAdmin MiddlewareFunc,
) {
	mux.Handle("GET /users", requireAdmin(http.HandlerFunc(h.ListUsers)))
	mux.Handle("GET /users/by-email/{email}", requireAdmin(http.HandlerFunc(h.GetUsersByEmail)))
	mux.Handle("DELETE /users/{id}", requireAdmin(http.HandlerFunc(h.DeleteUser)))

	mux.Handle("GET /users/{id}", requireSelfOrAdmin(http.HandlerFunc(h.GetUserById)))
	mux.Handle("PUT /users/{id}", requireSelfOrAdmin(http.HandlerFunc(h.UpdateUser)))

	mux.Handle("POST /users", requireAdmin(http.HandlerFunc(h.CreateUser)))
	mux.Handle("POST /users/with-profile", requireAdmin(http.HandlerFunc(h.CreateUserWithProfile)))
}
