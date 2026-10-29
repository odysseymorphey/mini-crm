package service

import (
	"context"

	"mini-crm/internal/domain"
)

type StatsService struct {
	statsRepo domain.StatsRepository
	teamRepo  domain.TeamRepository
}

func NewStatsService(statsRepo domain.StatsRepository, teamRepo domain.TeamRepository) *StatsService {
	return &StatsService{
		statsRepo: statsRepo,
		teamRepo:  teamRepo,
	}
}

func (s *StatsService) GetTeamStats(ctx context.Context, teamID, userID int64) (*domain.TeamStats, error) {
	member, err := s.teamRepo.GetMember(ctx, teamID, userID)
	if err != nil {
		return nil, domain.ErrForbidden
	}

	if member.Role != domain.RoleOwner && member.Role != domain.RoleAdmin {
		return nil, domain.ErrForbidden
	}

	return s.statsRepo.GetTeamStats(ctx, teamID)
}
