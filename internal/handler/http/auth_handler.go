package http

import (
	"encoding/json"
	"net/http"

	"mini-crm/internal/domain"
	"mini-crm/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register godoc
// @Summary Register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body domain.RegisterRequest true "User Registration"
// @Success 201 {object} domain.AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/v1/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authService.Register(r.Context(), req)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, resp)
}

// Login godoc
// @Summary Authenticate user and issue JWT
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body domain.LoginRequest true "User Login"
// @Success 200 {object} domain.AuthResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.authService.Login(r.Context(), req)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, resp)
}
