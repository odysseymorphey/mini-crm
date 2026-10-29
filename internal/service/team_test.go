package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"mini-crm/internal/domain"
	"mini-crm/internal/service"
)

type mockUserRepo struct {
	users map[int64]*domain.User
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	user.ID = int64(len(m.users) + 1)
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func TestTeamPermissions(t *testing.T) {
	ctx := context.Background()

	teamRepo := newMockTeamRepo()
	userRepo := &mockUserRepo{users: make(map[int64]*domain.User)}

	userRepo.users[1] = &domain.User{ID: 1, Name: "Owner", Email: "owner@team.com"}
	userRepo.users[2] = &domain.User{ID: 2, Name: "Admin", Email: "admin@team.com"}
	userRepo.users[3] = &domain.User{ID: 3, Name: "Member", Email: "member@team.com"}
	userRepo.users[4] = &domain.User{ID: 4, Name: "Candidate", Email: "candidate@team.com"}

	teamRepo.teams[1] = &domain.Team{ID: 1, Name: "Core", CreatedBy: 1, CreatedAt: time.Now()}
	teamRepo.members[teamMemberKey(1, 1)] = &domain.TeamMember{TeamID: 1, UserID: 1, Role: domain.RoleOwner}
	teamRepo.members[teamMemberKey(1, 2)] = &domain.TeamMember{TeamID: 1, UserID: 2, Role: domain.RoleAdmin}
	teamRepo.members[teamMemberKey(1, 3)] = &domain.TeamMember{TeamID: 1, UserID: 3, Role: domain.RoleMember}

	teamSvc := service.NewTeamService(teamRepo, userRepo)

	t.Run("Owner can invite an admin", func(t *testing.T) {
		candID := int64(4)
		mem, err := teamSvc.InviteMember(ctx, 1, 1, domain.InviteRequest{
			UserID: &candID,
			Role:   domain.RoleAdmin,
		})
		if err != nil {
			t.Fatalf("expected owner to be able to invite admin, got: %v", err)
		}
		if mem.Role != domain.RoleAdmin {
			t.Errorf("expected role to be admin, got %v", mem.Role)
		}
	})

	t.Run("Cannot invite with owner role", func(t *testing.T) {
		candID := int64(4)
		_, err := teamSvc.InviteMember(ctx, 1, 1, domain.InviteRequest{
			UserID: &candID,
			Role:   domain.RoleOwner,
		})
		if !errors.Is(err, domain.ErrCannotInviteOwner) {
			t.Fatalf("expected ErrCannotInviteOwner, got: %v", err)
		}
	})

	t.Run("Regular member cannot invite others", func(t *testing.T) {
		candID := int64(4)
		_, err := teamSvc.InviteMember(ctx, 1, 3, domain.InviteRequest{
			UserID: &candID,
			Role:   domain.RoleMember,
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected ErrForbidden when member invites, got: %v", err)
		}
	})

	t.Run("Cannot invite existing member", func(t *testing.T) {
		existingID := int64(2)
		_, err := teamSvc.InviteMember(ctx, 1, 1, domain.InviteRequest{
			UserID: &existingID,
			Role:   domain.RoleMember,
		})
		if !errors.Is(err, domain.ErrUserAlreadyInTeam) {
			t.Fatalf("expected ErrUserAlreadyInTeam, got: %v", err)
		}
	})
}
