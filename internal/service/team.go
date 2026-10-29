package service

import (
	"context"
	"strings"

	"mini-crm/internal/domain"
)

type TeamService struct {
	teamRepo domain.TeamRepository
	userRepo domain.UserRepository
}

func NewTeamService(teamRepo domain.TeamRepository, userRepo domain.UserRepository) *TeamService {
	return &TeamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

func (s *TeamService) CreateTeam(ctx context.Context, userID int64, req domain.CreateTeamRequest) (*domain.Team, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}

	team := &domain.Team{
		Name:      name,
		CreatedBy: userID,
	}

	member := &domain.TeamMember{
		UserID: userID,
		Role:   domain.RoleOwner,
	}

	if err := s.teamRepo.CreateTeamWithMember(ctx, team, member); err != nil {
		return nil, err
	}

	return team, nil
}

func (s *TeamService) GetUserTeams(ctx context.Context, userID int64) ([]domain.TeamWithRole, error) {
	return s.teamRepo.GetUserTeams(ctx, userID)
}

func (s *TeamService) InviteMember(ctx context.Context, teamID, requesterID int64, req domain.InviteRequest) (*domain.TeamMember, error) {
	requesterMember, err := s.teamRepo.GetMember(ctx, teamID, requesterID)
	if err != nil {
		return nil, domain.ErrForbidden
	}

	if requesterMember.Role != domain.RoleOwner && requesterMember.Role != domain.RoleAdmin {
		return nil, domain.ErrForbidden
	}

	if req.Role == domain.RoleOwner {
		return nil, domain.ErrCannotInviteOwner
	}

	if req.Role != domain.RoleAdmin && req.Role != domain.RoleMember {
		return nil, domain.ErrInvalidRole
	}

	var targetUserID int64
	if req.UserID != nil && *req.UserID > 0 {
		targetUserID = *req.UserID
		if _, err := s.userRepo.GetByID(ctx, targetUserID); err != nil {
			return nil, err
		}
	} else if req.Email != nil && strings.TrimSpace(*req.Email) != "" {
		email := strings.TrimSpace(strings.ToLower(*req.Email))
		user, err := s.userRepo.GetByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		targetUserID = user.ID
	} else {
		return nil, domain.ErrInvalidInput
	}

	isMember, err := s.teamRepo.IsMember(ctx, teamID, targetUserID)
	if err != nil {
		return nil, err
	}
	if isMember {
		return nil, domain.ErrUserAlreadyInTeam
	}

	newMember := &domain.TeamMember{
		TeamID: teamID,
		UserID: targetUserID,
		Role:   req.Role,
	}

	if err := s.teamRepo.AddMember(ctx, newMember); err != nil {
		return nil, err
	}

	return newMember, nil
}

func (s *TeamService) CheckMembership(ctx context.Context, teamID, userID int64) (*domain.TeamMember, error) {
	member, err := s.teamRepo.GetMember(ctx, teamID, userID)
	if err != nil {
		return nil, domain.ErrNotTeamMember
	}
	return member, nil
}
