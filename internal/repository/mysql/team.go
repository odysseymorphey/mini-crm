package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
	"mini-crm/internal/domain"
)

type TeamRepository struct {
	db *DB
}

func NewTeamRepository(db *DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) CreateTeamWithMember(ctx context.Context, team *domain.Team, member *domain.TeamMember) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	teamQuery := `INSERT INTO teams (name, created_by, created_at) VALUES (?, ?, NOW())`
	res, err := tx.ExecContext(ctx, teamQuery, team.Name, team.CreatedBy)
	if err != nil {
		return err
	}

	teamID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	team.ID = teamID

	memberQuery := `INSERT INTO team_members (team_id, user_id, role, created_at) VALUES (?, ?, ?, NOW())`
	_, err = tx.ExecContext(ctx, memberQuery, teamID, member.UserID, member.Role)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	member.TeamID = teamID
	return nil
}

func (r *TeamRepository) GetByID(ctx context.Context, id int64) (*domain.Team, error) {
	query := `SELECT id, name, created_by, created_at FROM teams WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var t domain.Team
	if err := row.Scan(&t.ID, &t.Name, &t.CreatedBy, &t.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTeamNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *TeamRepository) GetUserTeams(ctx context.Context, userID int64) ([]domain.TeamWithRole, error) {
	query := `
		SELECT t.id, t.name, t.created_by, t.created_at, tm.role
		FROM teams t
		JOIN team_members tm ON t.id = tm.team_id
		WHERE tm.user_id = ?
		ORDER BY t.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []domain.TeamWithRole
	for rows.Next() {
		var tr domain.TeamWithRole
		if err := rows.Scan(&tr.ID, &tr.Name, &tr.CreatedBy, &tr.CreatedAt, &tr.UserRole); err != nil {
			return nil, err
		}
		teams = append(teams, tr)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return teams, nil
}

func (r *TeamRepository) GetMember(ctx context.Context, teamID, userID int64) (*domain.TeamMember, error) {
	query := `SELECT team_id, user_id, role, created_at FROM team_members WHERE team_id = ? AND user_id = ?`
	row := r.db.QueryRowContext(ctx, query, teamID, userID)

	var m domain.TeamMember
	if err := row.Scan(&m.TeamID, &m.UserID, &m.Role, &m.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotTeamMember
		}
		return nil, err
	}
	return &m, nil
}

func (r *TeamRepository) AddMember(ctx context.Context, member *domain.TeamMember) error {
	query := `INSERT INTO team_members (team_id, user_id, role, created_at) VALUES (?, ?, ?, NOW())`
	_, err := r.db.ExecContext(ctx, query, member.TeamID, member.UserID, member.Role)
	if err != nil {
		var mySQLErr *mysql.MySQLError
		if errors.As(err, &mySQLErr) && mySQLErr.Number == 1062 {
			return domain.ErrUserAlreadyInTeam
		}
		return err
	}
	return nil
}

func (r *TeamRepository) IsMember(ctx context.Context, teamID, userID int64) (bool, error) {
	query := `SELECT 1 FROM team_members WHERE team_id = ? AND user_id = ? LIMIT 1`
	var exists int
	err := r.db.QueryRowContext(ctx, query, teamID, userID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
