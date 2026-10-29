package domain

import (
	"context"
	"time"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

func (r Role) IsValid() bool {
	return r == RoleOwner || r == RoleAdmin || r == RoleMember
}

type Team struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type TeamMember struct {
	TeamID    int64     `json:"team_id"`
	UserID    int64     `json:"user_id"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	User      *User     `json:"user,omitempty"`
}

type TeamWithRole struct {
	Team
	UserRole Role `json:"user_role"`
}

type CreateTeamRequest struct {
	Name string `json:"name"`
}

type InviteRequest struct {
	UserID *int64  `json:"user_id,omitempty"`
	Email  *string `json:"email,omitempty"`
	Role   Role    `json:"role"`
}

type TeamRepository interface {
	CreateTeamWithMember(ctx context.Context, team *Team, member *TeamMember) error
	GetByID(ctx context.Context, id int64) (*Team, error)
	GetUserTeams(ctx context.Context, userID int64) ([]TeamWithRole, error)
	GetMember(ctx context.Context, teamID, userID int64) (*TeamMember, error)
	AddMember(ctx context.Context, member *TeamMember) error
	IsMember(ctx context.Context, teamID, userID int64) (bool, error)
}
