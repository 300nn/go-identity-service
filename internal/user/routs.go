package user

import (
	"net/http"
)

func (h *Handler) RegisterRouts(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler) {
	mux.Handle("GET /users", requireAuth(http.HandlerFunc(h.ListUsers)))
	mux.Handle("GET /users/{id}", requireAuth(http.HandlerFunc(h.GetUserById)))
	mux.Handle("PUT /users/{id}", requireAuth(http.HandlerFunc(h.UpdateUser)))
	mux.Handle("DELETE /users/{id}", requireAuth(http.HandlerFunc(h.DeleteUser)))
	mux.Handle("GET /users/by-email/{email}", requireAuth(http.HandlerFunc(h.GetUsersByEmail)))

	mux.Handle("POST /users", requireAuth(http.HandlerFunc(h.CreateUser)))

	mux.Handle("POST /users/with-profile", requireAuth(http.HandlerFunc(h.CreateUserWithProfile)))
}
