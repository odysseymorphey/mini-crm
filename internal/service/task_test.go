package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"mini-crm/internal/domain"
	"mini-crm/internal/service"
)

type mockTeamRepo struct {
	members   map[string]*domain.TeamMember
	teams     map[int64]*domain.Team
	userTeams map[int64][]domain.TeamWithRole
}

func newMockTeamRepo() *mockTeamRepo {
	return &mockTeamRepo{
		members:   make(map[string]*domain.TeamMember),
		teams:     make(map[int64]*domain.Team),
		userTeams: make(map[int64][]domain.TeamWithRole),
	}
}

func (m *mockTeamRepo) CreateTeamWithMember(ctx context.Context, team *domain.Team, member *domain.TeamMember) error {
	team.ID = 1
	member.TeamID = 1
	m.teams[1] = team
	m.members["1:1"] = member
	return nil
}

func (m *mockTeamRepo) GetByID(ctx context.Context, id int64) (*domain.Team, error) {
	if t, ok := m.teams[id]; ok {
		return t, nil
	}
	return nil, domain.ErrTeamNotFound
}

func (m *mockTeamRepo) GetUserTeams(ctx context.Context, userID int64) ([]domain.TeamWithRole, error) {
	return m.userTeams[userID], nil
}

func (m *mockTeamRepo) GetMember(ctx context.Context, teamID, userID int64) (*domain.TeamMember, error) {
	key := teamMemberKey(teamID, userID)
	if mem, ok := m.members[key]; ok {
		return mem, nil
	}
	return nil, domain.ErrNotTeamMember
}

func (m *mockTeamRepo) AddMember(ctx context.Context, member *domain.TeamMember) error {
	key := teamMemberKey(member.TeamID, member.UserID)
	if _, exists := m.members[key]; exists {
		return domain.ErrUserAlreadyInTeam
	}
	m.members[key] = member
	return nil
}

func (m *mockTeamRepo) IsMember(ctx context.Context, teamID, userID int64) (bool, error) {
	key := teamMemberKey(teamID, userID)
	_, ok := m.members[key]
	return ok, nil
}

func teamMemberKey(teamID, userID int64) string {
	return string(rune(teamID)) + ":" + string(rune(userID))
}

type mockTaskRepo struct {
	tasks map[int64]*domain.Task
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{
		tasks: make(map[int64]*domain.Task),
	}
}

func (m *mockTaskRepo) CreateWithHistory(ctx context.Context, task *domain.Task, history *domain.TaskHistory) error {
	task.ID = int64(len(m.tasks) + 1)
	task.Version = 1
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepo) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	if t, ok := m.tasks[id]; ok {
		cp := *t
		return &cp, nil
	}
	return nil, domain.ErrTaskNotFound
}

func (m *mockTaskRepo) GetByIDForUpdate(ctx context.Context, id int64) (*domain.Task, error) {
	return m.GetByID(ctx, id)
}

func (m *mockTaskRepo) UpdateWithHistory(ctx context.Context, task *domain.Task, history *domain.TaskHistory) error {
	existing, ok := m.tasks[task.ID]
	if !ok {
		return domain.ErrTaskNotFound
	}
	if existing.Version != task.Version {
		return domain.ErrVersionConflict
	}
	task.Version++
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepo) List(ctx context.Context, filter domain.TaskFilter) (*domain.TaskListResponse, error) {
	var res []domain.Task
	for _, t := range m.tasks {
		if t.TeamID == filter.TeamID {
			res = append(res, *t)
		}
	}
	return &domain.TaskListResponse{Tasks: res, TotalCount: int64(len(res))}, nil
}

type mockHistoryRepo struct {
	histories []domain.TaskHistory
}

func (m *mockHistoryRepo) Create(ctx context.Context, history *domain.TaskHistory) error {
	m.histories = append(m.histories, *history)
	return nil
}

func (m *mockHistoryRepo) GetByTaskID(ctx context.Context, taskID int64) ([]domain.TaskHistory, error) {
	return m.histories, nil
}

func setupTaskServiceTest() (*service.TaskService, *mockTeamRepo, *mockTaskRepo) {
	teamRepo := newMockTeamRepo()
	taskRepo := newMockTaskRepo()
	histRepo := &mockHistoryRepo{}

	teamRepo.teams[1] = &domain.Team{ID: 1, Name: "Engineering", CreatedBy: 1, CreatedAt: time.Now()}

	teamRepo.members[teamMemberKey(1, 1)] = &domain.TeamMember{TeamID: 1, UserID: 1, Role: domain.RoleOwner}
	teamRepo.members[teamMemberKey(1, 2)] = &domain.TeamMember{TeamID: 1, UserID: 2, Role: domain.RoleAdmin}
	teamRepo.members[teamMemberKey(1, 3)] = &domain.TeamMember{TeamID: 1, UserID: 3, Role: domain.RoleMember}
	teamRepo.members[teamMemberKey(1, 4)] = &domain.TeamMember{TeamID: 1, UserID: 4, Role: domain.RoleMember}
	teamRepo.members[teamMemberKey(1, 5)] = &domain.TeamMember{TeamID: 1, UserID: 5, Role: domain.RoleMember}

	assignee4 := int64(4)
	taskRepo.tasks[10] = &domain.Task{
		ID:          10,
		TeamID:      1,
		Title:       "Original Title",
		Description: "Original Description",
		Status:      domain.StatusTodo,
		CreatedBy:   3,
		AssigneeID:  &assignee4,
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	taskService := service.NewTaskService(taskRepo, teamRepo, histRepo, nil)
	return taskService, teamRepo, taskRepo
}

func TestTaskPermissions(t *testing.T) {
	ctx := context.Background()

	t.Run("Owner can edit any task fully", func(t *testing.T) {
		svc, _, _ := setupTaskServiceTest()
		newTitle := "Updated by Owner"
		newStatus := domain.StatusInProgress
		newAssignee := int64(5)

		task, err := svc.UpdateTask(ctx, 10, 1, domain.UpdateTaskRequest{
			Title:      &newTitle,
			Status:     &newStatus,
			AssigneeID: &newAssignee,
			Version:    1,
		})
		if err != nil {
			t.Fatalf("expected owner update to succeed, got: %v", err)
		}
		if task.Title != newTitle || task.Status != newStatus || *task.AssigneeID != 5 {
			t.Errorf("task fields were not updated properly")
		}
	})

	t.Run("Admin can edit any task fully", func(t *testing.T) {
		svc, _, _ := setupTaskServiceTest()
		newTitle := "Updated by Admin"
		newDesc := "New description by admin"

		task, err := svc.UpdateTask(ctx, 10, 2, domain.UpdateTaskRequest{
			Title:       &newTitle,
			Description: &newDesc,
			Version:     1,
		})
		if err != nil {
			t.Fatalf("expected admin update to succeed, got: %v", err)
		}
		if task.Title != newTitle || task.Description != newDesc {
			t.Errorf("task fields were not updated properly")
		}
	})

	t.Run("Creator (regular member) can edit their task", func(t *testing.T) {
		svc, _, _ := setupTaskServiceTest()
		newTitle := "Updated by Creator"
		newAssignee := int64(5)

		task, err := svc.UpdateTask(ctx, 10, 3, domain.UpdateTaskRequest{
			Title:      &newTitle,
			AssigneeID: &newAssignee,
			Version:    1,
		})
		if err != nil {
			t.Fatalf("expected creator update to succeed, got: %v", err)
		}
		if task.Title != newTitle || *task.AssigneeID != 5 {
			t.Errorf("task fields were not updated properly")
		}
	})

	t.Run("Creator cannot assign task to non-team member", func(t *testing.T) {
		svc, _, _ := setupTaskServiceTest()
		nonMemberID := int64(99)

		_, err := svc.UpdateTask(ctx, 10, 3, domain.UpdateTaskRequest{
			AssigneeID: &nonMemberID,
			Version:    1,
		})
		if !errors.Is(err, domain.ErrAssigneeNotMember) {
			t.Fatalf("expected ErrAssigneeNotMember, got: %v", err)
		}
	})

	t.Run("Assignee can update status", func(t *testing.T) {
		svc, _, _ := setupTaskServiceTest()
		newStatus := domain.StatusDone

		task, err := svc.UpdateTask(ctx, 10, 4, domain.UpdateTaskRequest{
			Status:  &newStatus,
			Version: 1,
		})
		if err != nil {
			t.Fatalf("expected assignee to be able to change status, got: %v", err)
		}
		if task.Status != domain.StatusDone {
			t.Errorf("expected status to be done, got %v", task.Status)
		}
	})

	t.Run("Assignee cannot change title", func(t *testing.T) {
		svc, _, _ := setupTaskServiceTest()
		forbiddenTitle := "Assignee Trying To Rename"

		_, err := svc.UpdateTask(ctx, 10, 4, domain.UpdateTaskRequest{
			Title:   &forbiddenTitle,
			Version: 1,
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected ErrForbidden when assignee changes title, got: %v", err)
		}
	})

	t.Run("Assignee cannot reassign task", func(t *testing.T) {
		svc, _, _ := setupTaskServiceTest()
		newAssignee := int64(5)

		_, err := svc.UpdateTask(ctx, 10, 4, domain.UpdateTaskRequest{
			AssigneeID: &newAssignee,
			Version:    1,
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected ErrForbidden when assignee reassigns task, got: %v", err)
		}
	})

	t.Run("Other member cannot edit task", func(t *testing.T) {
		svc, _, _ := setupTaskServiceTest()
		newStatus := domain.StatusDone

		_, err := svc.UpdateTask(ctx, 10, 5, domain.UpdateTaskRequest{
			Status:  &newStatus,
			Version: 1,
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected ErrForbidden for non-involved member, got: %v", err)
		}
	})

	t.Run("Non-member cannot view or edit task", func(t *testing.T) {
		svc, _, _ := setupTaskServiceTest()
		newStatus := domain.StatusDone

		_, err := svc.UpdateTask(ctx, 10, 99, domain.UpdateTaskRequest{
			Status:  &newStatus,
			Version: 1,
		})
		if !errors.Is(err, domain.ErrNotTeamMember) {
			t.Fatalf("expected ErrNotTeamMember for outsider, got: %v", err)
		}
	})

	t.Run("Optimistic locking rejects stale version", func(t *testing.T) {
		svc, _, _ := setupTaskServiceTest()
		newStatus := domain.StatusInProgress

		_, err := svc.UpdateTask(ctx, 10, 1, domain.UpdateTaskRequest{
			Status:  &newStatus,
			Version: 999,
		})
		if !errors.Is(err, domain.ErrVersionConflict) {
			t.Fatalf("expected ErrVersionConflict, got: %v", err)
		}
	})
}
