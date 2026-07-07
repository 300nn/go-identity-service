package user

import "net/http"

func (h *Handler) RegisterRouts(mux *http.ServeMux) {
	mux.HandleFunc("POST /users", h.CreateUser)
	mux.HandleFunc("GET /users", h.ListUsers)
	mux.HandleFunc("GET /users/{id}", h.GetUserById)
	mux.HandleFunc("PUT /users/{id}", h.UpdateUser)
	mux.HandleFunc("DELETE /users/{id}", h.DeleteUser)
	mux.HandleFunc("GET /users/by-email/{email}", h.GetUsersByEmail)
}
