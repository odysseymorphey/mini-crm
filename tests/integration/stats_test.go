package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"mini-crm/internal/config"
	"mini-crm/internal/domain"
	mysqlrepo "mini-crm/internal/repository/mysql"
	"mini-crm/internal/service"
)

func getTestDBConfig() config.DatabaseConfig {
	cfg := config.Load().Database
	if host := os.Getenv("TEST_DB_HOST"); host != "" {
		cfg.Host = host
	}
	if port := os.Getenv("TEST_DB_PORT"); port != "" {
		cfg.Port = port
	}
	return cfg
}

func TestTeamStatsIntegration(t *testing.T) {
	dbCfg := getTestDBConfig()

	db, err := mysqlrepo.NewDB(dbCfg)
	if err != nil {
		t.Skipf("skipping integration test: cannot connect to MySQL database at %s:%s: %v", dbCfg.Host, dbCfg.Port, err)
		return
	}
	defer db.Close()

	if err := db.RunMigrations("../../migrations/001_init.sql"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	ctx := context.Background()

	userRepo := mysqlrepo.NewUserRepository(db)
	teamRepo := mysqlrepo.NewTeamRepository(db)
	commentRepo := mysqlrepo.NewCommentRepository(db)
	statsRepo := mysqlrepo.NewStatsRepository(db)
	statsService := service.NewStatsService(statsRepo, teamRepo)

	testSuffix := time.Now().UnixNano()

	owner := &domain.User{Email: fmt.Sprintf("owner_%d@test.com", testSuffix), PasswordHash: "hash", Name: "Alice Owner"}
	if err := userRepo.Create(ctx, owner); err != nil {
		t.Fatalf("failed to create owner: %v", err)
	}

	admin := &domain.User{Email: fmt.Sprintf("admin_%d@test.com", testSuffix), PasswordHash: "hash", Name: "Bob Admin"}
	if err := userRepo.Create(ctx, admin); err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	member1 := &domain.User{Email: fmt.Sprintf("member1_%d@test.com", testSuffix), PasswordHash: "hash", Name: "Charlie Dev"}
	if err := userRepo.Create(ctx, member1); err != nil {
		t.Fatalf("failed to create member1: %v", err)
	}

	member2 := &domain.User{Email: fmt.Sprintf("member2_%d@test.com", testSuffix), PasswordHash: "hash", Name: "Diana QA"}
	if err := userRepo.Create(ctx, member2); err != nil {
		t.Fatalf("failed to create member2: %v", err)
	}

	outsider := &domain.User{Email: fmt.Sprintf("outsider_%d@test.com", testSuffix), PasswordHash: "hash", Name: "Eve Outsider"}
	if err := userRepo.Create(ctx, outsider); err != nil {
		t.Fatalf("failed to create outsider: %v", err)
	}

	team := &domain.Team{Name: "Core Platform", CreatedBy: owner.ID}
	ownerMember := &domain.TeamMember{UserID: owner.ID, Role: domain.RoleOwner}
	if err := teamRepo.CreateTeamWithMember(ctx, team, ownerMember); err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	if err := teamRepo.AddMember(ctx, &domain.TeamMember{TeamID: team.ID, UserID: admin.ID, Role: domain.RoleAdmin}); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}
	if err := teamRepo.AddMember(ctx, &domain.TeamMember{TeamID: team.ID, UserID: member1.ID, Role: domain.RoleMember}); err != nil {
		t.Fatalf("failed to add member1: %v", err)
	}
	if err := teamRepo.AddMember(ctx, &domain.TeamMember{TeamID: team.ID, UserID: member2.ID, Role: domain.RoleMember}); err != nil {
		t.Fatalf("failed to add member2: %v", err)
	}

	now := time.Now()

	created1 := now.Add(-5*24*time.Hour - 10*time.Hour)
	closed1 := now.Add(-5 * 24 * time.Hour)
	res1, err := db.ExecContext(ctx, `
		INSERT INTO tasks (team_id, title, status, created_by, assignee_id, created_at, updated_at, closed_at, version)
		VALUES (?, 'Task 1', 'closed', ?, ?, ?, ?, ?, 1)
	`, team.ID, owner.ID, member1.ID, created1, closed1, closed1)
	if err != nil {
		t.Fatalf("failed to insert task 1: %v", err)
	}
	task1ID, _ := res1.LastInsertId()

	created2 := now.Add(-2*24*time.Hour - 5*time.Hour)
	closed2 := now.Add(-2 * 24 * time.Hour)
	_, err = db.ExecContext(ctx, `
		INSERT INTO tasks (team_id, title, status, created_by, assignee_id, created_at, updated_at, closed_at, version)
		VALUES (?, 'Task 2', 'closed', ?, ?, ?, ?, ?, 1)
	`, team.ID, owner.ID, member1.ID, created2, closed2, closed2)
	if err != nil {
		t.Fatalf("failed to insert task 2: %v", err)
	}

	created3 := now.Add(-1*24*time.Hour - 2*time.Hour)
	closed3 := now.Add(-1 * 24 * time.Hour)
	_, err = db.ExecContext(ctx, `
		INSERT INTO tasks (team_id, title, status, created_by, assignee_id, created_at, updated_at, closed_at, version)
		VALUES (?, 'Task 3', 'closed', ?, ?, ?, ?, ?, 1)
	`, team.ID, owner.ID, member2.ID, created3, closed3, closed3)
	if err != nil {
		t.Fatalf("failed to insert task 3: %v", err)
	}

	created4 := now.Add(-40*24*time.Hour - 4*time.Hour)
	closed4 := now.Add(-40 * 24 * time.Hour)
	_, err = db.ExecContext(ctx, `
		INSERT INTO tasks (team_id, title, status, created_by, assignee_id, created_at, updated_at, closed_at, version)
		VALUES (?, 'Task 4', 'closed', ?, ?, ?, ?, ?, 1)
	`, team.ID, owner.ID, member2.ID, created4, closed4, closed4)
	if err != nil {
		t.Fatalf("failed to insert task 4: %v", err)
	}

	res5, err := db.ExecContext(ctx, `
		INSERT INTO tasks (team_id, title, status, created_by, assignee_id, created_at, updated_at, closed_at, version)
		VALUES (?, 'Task 5', 'todo', ?, ?, NOW(), NOW(), NULL, 1)
	`, team.ID, owner.ID, member1.ID)
	if err != nil {
		t.Fatalf("failed to insert task 5: %v", err)
	}
	task5ID, _ := res5.LastInsertId()

	_, err = db.ExecContext(ctx, `
		INSERT INTO tasks (team_id, title, status, created_by, assignee_id, created_at, updated_at, closed_at, version)
		VALUES (?, 'Task 6', 'in_progress', ?, ?, NOW(), NOW(), NULL, 1)
	`, team.ID, owner.ID, member2.ID)
	if err != nil {
		t.Fatalf("failed to insert task 6: %v", err)
	}

	otherTeam := &domain.Team{Name: "Other Team", CreatedBy: outsider.ID}
	_ = teamRepo.CreateTeamWithMember(ctx, otherTeam, &domain.TeamMember{UserID: outsider.ID, Role: domain.RoleOwner})
	_, _ = db.ExecContext(ctx, `
		INSERT INTO tasks (team_id, title, status, created_by, assignee_id, created_at, updated_at, closed_at, version)
		VALUES (?, 'Other Task', 'closed', ?, ?, NOW(), NOW(), NOW(), 1)
	`, otherTeam.ID, outsider.ID, outsider.ID)

	if err := commentRepo.Create(ctx, &domain.TaskComment{TaskID: task1ID, UserID: member1.ID, Content: "Comment 1"}); err != nil {
		t.Fatalf("failed to create comment: %v", err)
	}
	if err := commentRepo.Create(ctx, &domain.TaskComment{TaskID: task1ID, UserID: member2.ID, Content: "Comment 2"}); err != nil {
		t.Fatalf("failed to create comment: %v", err)
	}
	if err := commentRepo.Create(ctx, &domain.TaskComment{TaskID: task1ID, UserID: admin.ID, Content: "Comment 3"}); err != nil {
		t.Fatalf("failed to create comment: %v", err)
	}

	if err := commentRepo.Create(ctx, &domain.TaskComment{TaskID: task5ID, UserID: member1.ID, Content: "Comment 4"}); err != nil {
		t.Fatalf("failed to create comment: %v", err)
	}
	if err := commentRepo.Create(ctx, &domain.TaskComment{TaskID: task5ID, UserID: owner.ID, Content: "Comment 5"}); err != nil {
		t.Fatalf("failed to create comment: %v", err)
	}

	stats, err := statsRepo.GetTeamStats(ctx, team.ID)
	if err != nil {
		t.Fatalf("GetTeamStats failed: %v", err)
	}

	if stats.TotalComments != 5 {
		t.Errorf("expected 5 comments on team tasks, got %d", stats.TotalComments)
	}

	statusMap := make(map[string]int64)
	for _, sc := range stats.TasksByStatus {
		statusMap[sc.Status] = sc.Count
	}
	if statusMap["closed"] != 4 {
		t.Errorf("expected 4 closed tasks, got %d", statusMap["closed"])
	}
	if statusMap["todo"] != 1 {
		t.Errorf("expected 1 todo task, got %d", statusMap["todo"])
	}
	if statusMap["in_progress"] != 1 {
		t.Errorf("expected 1 in_progress task, got %d", statusMap["in_progress"])
	}

	if len(stats.TopAssignees) < 2 {
		t.Fatalf("expected at least 2 top assignees, got %d", len(stats.TopAssignees))
	}
	if stats.TopAssignees[0].UserID != member1.ID || stats.TopAssignees[0].ClosedCount != 2 {
		t.Errorf("expected top assignee to be member1 with 2 closed tasks, got: %+v", stats.TopAssignees[0])
	}
	if stats.TopAssignees[1].UserID != member2.ID || stats.TopAssignees[1].ClosedCount != 1 {
		t.Errorf("expected second assignee to be member2 with 1 closed task, got: %+v", stats.TopAssignees[1])
	}

	if stats.AvgClosingTimeSeconds <= 0 {
		t.Errorf("expected avg closing time > 0, got %f", stats.AvgClosingTimeSeconds)
	}

	t.Run("Owner access to stats allowed", func(t *testing.T) {
		res, err := statsService.GetTeamStats(ctx, team.ID, owner.ID)
		if err != nil {
			t.Fatalf("expected owner to access stats, got: %v", err)
		}
		if res.TotalComments != 5 {
			t.Errorf("unexpected total comments: %d", res.TotalComments)
		}
	})

	t.Run("Admin access to stats allowed", func(t *testing.T) {
		res, err := statsService.GetTeamStats(ctx, team.ID, admin.ID)
		if err != nil {
			t.Fatalf("expected admin to access stats, got: %v", err)
		}
		if res.TotalComments != 5 {
			t.Errorf("unexpected total comments: %d", res.TotalComments)
		}
	})

	t.Run("Member access to stats forbidden", func(t *testing.T) {
		_, err := statsService.GetTeamStats(ctx, team.ID, member1.ID)
		if err == nil {
			t.Fatalf("expected error for regular member, got nil")
		}
	})

	t.Run("Outsider access to stats forbidden", func(t *testing.T) {
		_, err := statsService.GetTeamStats(ctx, team.ID, outsider.ID)
		if err == nil {
			t.Fatalf("expected error for outsider, got nil")
		}
	})
}
