package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/SergeyRG/gofermart/internal/model"
	"github.com/SergeyRG/gofermart/internal/services"
	"github.com/jackc/pgx/v5/pgconn"
)

type PSQLUserRepo struct {
	BaseRepository
}

func NewPSQLUserRepo(db *sql.DB) *PSQLUserRepo {
	return &PSQLUserRepo{
		BaseRepository: BaseRepository{db: db},
	}
}

func (repo *PSQLUserRepo) GetUserByLogin(ctx context.Context, login string) (*model.User, error) {
	qe := repo.GetExecutor(ctx)
	query := `SELECT user_id, login, pwd_hash
			FROM users
			WHERE login = $1`

	sqlResults := qe.QueryRowContext(ctx, query, login)

	user := model.User{}
	err := sqlResults.Scan(&user.UserID, &user.Login, &user.PwdHash)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, services.ErrWrongLoginOrPassword
		}
		return nil, fmt.Errorf(
			"ошибка получения пользователя из БД: %w", err)
	}

	return &user, nil
}

func (repo *PSQLUserRepo) AddUser(ctx context.Context, user model.User) (model.UserID, error) {
	qe := repo.GetExecutor(ctx)
	query := `
			INSERT INTO users (login, pwd_hash)
			VALUES ($1, $2) 
			RETURNING user_id;`

	sqlResults := qe.QueryRowContext(ctx, query, user.Login, user.PwdHash)

	var userID model.UserID
	err := sqlResults.Scan(&userID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, services.ErrLoginBusy
		}
		return 0, fmt.Errorf(
			"ошибка сохранения пользователя в БД: %w", err)
	}

	return userID, nil
}
