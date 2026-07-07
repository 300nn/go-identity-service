package user

import (
	"CrudTutorialProject/internal/response"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	req, ok := response.DecodeJSON[CreateUserRequest](w, r)
	if !ok {
		return
	}

	created, err := h.service.CreateUser(r.Context(), CreateUserInput{
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
	})

	if err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusCreated, ToResponse(created))
}

func (h *Handler) GetUserById(w http.ResponseWriter, r *http.Request) {
	id, ok := parseId(w, r)

	if !ok {
		return
	}

	found, err := h.service.GetUser(r.Context(), id)

	if err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusOK, ToResponse(found))
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())

	if err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusOK, ToResponseList(users))
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseId(w, r)
	if !ok {
		return
	}

	req, ok := response.DecodeJSON[UpdateUserRequest](w, r)
	if !ok {
		return
	}

	updated, err := h.service.UpdateUser(r.Context(), id, UpdateUserInput{
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
	})
	if err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusOK, ToResponse(updated))
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseId(w, r)
	if !ok {
		return
	}

	if err := h.service.DeleteUser(r.Context(), id); err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) GetUsersByEmail(w http.ResponseWriter, r *http.Request) {
	email, ok := parseString(w, r, "email")

	if !ok {
		return
	}
	user, err := h.service.GetUserByEmail(r.Context(), email)

	if err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusOK, ToResponse(user))
}

func parseString(w http.ResponseWriter, r *http.Request, str string) (string, bool) {
	res := r.PathValue(str)

	res = strings.TrimSpace(res)

	if res == "" {
		response.SendValidationError(w, map[string]string{
			str: "Required in path variable",
		})
		return "", false
	}

	return res, true
}

func parseId(w http.ResponseWriter, r *http.Request) (int64, bool) {
	rawId := r.PathValue("id")

	id, err := strconv.Atoi(rawId)
	if err != nil {
		response.SendValidationError(w, map[string]string{
			"id": "Required in path variable or id invalid",
		})
		return 0, false
	}

	return int64(id), true
}
