package domain

import "errors"

var (
	ErrNotFound            = errors.New("resource not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrTeamNotFound        = errors.New("team not found")
	ErrTaskNotFound        = errors.New("task not found")
	ErrEmailAlreadyExists  = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden: insufficient permissions")
	ErrNotTeamMember       = errors.New("user is not a member of the team")
	ErrAssigneeNotMember   = errors.New("assignee must be a member of the same team")
	ErrCannotInviteOwner   = errors.New("cannot invite member with owner role")
	ErrUserAlreadyInTeam   = errors.New("user is already a member of this team")
	ErrInvalidRole         = errors.New("invalid role specified")
	ErrCannotDemoteOwner   = errors.New("cannot remove or change role of team owner")
	ErrVersionConflict     = errors.New("conflict: task version mismatch (concurrent update)")
	ErrInvalidInput        = errors.New("invalid input data")
	ErrMissingTeamID       = errors.New("team_id query parameter is required")
)
