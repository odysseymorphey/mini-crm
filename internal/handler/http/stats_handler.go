package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"mini-crm/internal/middleware"
	"mini-crm/internal/service"
)

type StatsHandler struct {
	statsService *service.StatsService
}

func NewStatsHandler(statsService *service.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

// GetTeamStats godoc
// @Summary SQL report and metrics for a specific team (owner or admin only)
// @Tags Analytics
// @Security BearerAuth
// @Produce json
// @Param team_id path int true "Team ID"
// @Success 200 {object} domain.TeamStats
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/v1/teams/{team_id}/stats [get]
func (h *StatsHandler) GetTeamStats(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	teamIDParam := chi.URLParam(r, "team_id")
	teamID, err := strconv.ParseInt(teamIDParam, 10, 64)
	if err != nil || teamID <= 0 {
		RespondError(w, http.StatusBadRequest, "invalid team_id parameter")
		return
	}

	stats, err := h.statsService.GetTeamStats(r.Context(), teamID, userID)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, stats)
}
