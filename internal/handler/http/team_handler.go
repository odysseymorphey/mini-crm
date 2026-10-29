package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"mini-crm/internal/domain"
	"mini-crm/internal/middleware"
	"mini-crm/internal/service"
)

type TeamHandler struct {
	teamService *service.TeamService
}

func NewTeamHandler(teamService *service.TeamService) *TeamHandler {
	return &TeamHandler{teamService: teamService}
}

// CreateTeam godoc
// @Summary Create a team
// @Tags Teams
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body domain.CreateTeamRequest true "Team creation"
// @Success 201 {object} domain.Team
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/teams [post]
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	var req domain.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	team, err := h.teamService.CreateTeam(r.Context(), userID, req)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, team)
}

// GetUserTeams godoc
// @Summary List current user teams
// @Tags Teams
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.TeamWithRole
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/teams [get]
func (h *TeamHandler) GetUserTeams(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	teams, err := h.teamService.GetUserTeams(r.Context(), userID)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	if teams == nil {
		teams = []domain.TeamWithRole{}
	}

	RespondJSON(w, http.StatusOK, teams)
}

// InviteMember godoc
// @Summary Invite a member to the team (owner or admin only)
// @Tags Teams
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Team ID"
// @Param request body domain.InviteRequest true "Invite payload"
// @Success 201 {object} domain.TeamMember
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/v1/teams/{id}/invite [post]
func (h *TeamHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	idParam := chi.URLParam(r, "id")
	teamID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || teamID <= 0 {
		RespondError(w, http.StatusBadRequest, "invalid team ID")
		return
	}

	var req domain.InviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	member, err := h.teamService.InviteMember(r.Context(), teamID, userID, req)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, member)
}
