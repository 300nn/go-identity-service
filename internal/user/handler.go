package user

import (
	"CrudTutorialProject/internal/apperror"
	"CrudTutorialProject/internal/response"
	"CrudTutorialProject/internal/validation"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
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

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	req, ok := response.DecodeJSON[CreateUserRequest](w, r)
	if !ok {
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.HandleError(w, h.logger, err)
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
	req, err := h.ParseListUsersRequest(r)

	if err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	result, err := h.service.ListUsers(r.Context(), ListUsersInput{
		Limit:  req.Limit,
		Offset: req.Offset,
		Email:  req.Email,
		Sort:   req.Sort,
	})

	if err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusOK, response.Page[UserResponse]{
		Items: ToResponseList(result.Users),
		Pagination: response.Pagination{
			Limit:  req.Limit,
			Offset: req.Offset,
			Total:  result.Total,
		},
	})
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

	if err := h.validator.Struct(req); err != nil {
		response.HandleError(w, h.logger, err)
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

func (h *Handler) CreateUserWithProfile(w http.ResponseWriter, r *http.Request) {
	req, ok := response.DecodeJSON[CreateUserWithProfileRequest](w, r)
	if !ok {
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	created, err := h.service.CreateUserWithProfile(r.Context(), CreateUserWithProfileInput{
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
		Bio:   req.Bio,
	})

	if err != nil {
		response.HandleError(w, h.logger, err)
		return
	}

	response.JSON(w, http.StatusCreated, UserWithProfileResponse{
		User:    ToResponse(created.User),
		Profile: ProfileToResponse(created.Profile),
	})
}

func (h *Handler) ParseListUsersRequest(r *http.Request) (ListUsersRequest, error) {
	query := r.URL.Query()

	fields := make(map[string]string)

	limit := parseIntQuery(query.Get("limit"), "limit", 20, fields)
	offset := parseIntQuery(query.Get("offset"), "offset", 0, fields)

	sortValue := strings.TrimSpace(query.Get("sort"))

	if sortValue == "" {
		sortValue = "id_asc"
	}

	req := ListUsersRequest{
		Limit:  limit,
		Offset: offset,
		Email:  query.Get("email"),
		Sort:   sortValue,
	}

	if len(fields) > 0 {
		return req, apperror.NewFieldValidationError(fields)
	}

	if err := h.validator.Struct(req); err != nil {
		return req, err
	}

	return req, nil
}

func parseIntQuery(raw string, fieldName string, defaultValue int, fields map[string]string) int {
	raw = strings.TrimSpace(raw)

	if raw == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(raw)

	if err != nil {
		fields[fieldName] = "must be a number"
		return defaultValue
	}

	return value
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
