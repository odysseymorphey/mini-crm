package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"mini-crm/internal/domain"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func RespondJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("error encoding json response", "err", err)
		}
	}
}

func RespondError(w http.ResponseWriter, statusCode int, message string) {
	RespondJSON(w, statusCode, ErrorResponse{Error: message})
}

func HandleDomainError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, domain.ErrNotFound),
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrTeamNotFound),
		errors.Is(err, domain.ErrTaskNotFound):
		RespondError(w, http.StatusNotFound, err.Error())

	case errors.Is(err, domain.ErrUnauthorized),
		errors.Is(err, domain.ErrInvalidCredentials):
		RespondError(w, http.StatusUnauthorized, err.Error())

	case errors.Is(err, domain.ErrForbidden),
		errors.Is(err, domain.ErrNotTeamMember),
		errors.Is(err, domain.ErrCannotDemoteOwner):
		RespondError(w, http.StatusForbidden, err.Error())

	case errors.Is(err, domain.ErrCannotInviteOwner),
		errors.Is(err, domain.ErrInvalidRole),
		errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrMissingTeamID),
		errors.Is(err, domain.ErrAssigneeNotMember):
		RespondError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, domain.ErrEmailAlreadyExists),
		errors.Is(err, domain.ErrUserAlreadyInTeam),
		errors.Is(err, domain.ErrVersionConflict):
		RespondError(w, http.StatusConflict, err.Error())

	default:
		slog.Error("internal server error", "err", err)
		RespondError(w, http.StatusInternalServerError, "internal server error")
	}
}
