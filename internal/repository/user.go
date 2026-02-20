package repository

import (
	"context"
	"database/sql"
)

type User struct {
	ID       int64
	TgUserID int64
	Username sql.NullString
	FullName sql.NullString
	Phone    sql.NullString
}

type UsersRepo struct {
	db *sql.DB
}

func NewUsersRepo(db *sql.DB) *UsersRepo { return &UsersRepo{db: db} }

func (r *UsersRepo) ExistsByTgUserID(ctx context.Context, tgUserID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`select exists(select 1 from users where tg_user_id = $1)`,
		tgUserID,
	).Scan(&exists)
	return exists, err
}

func (r *UsersRepo) UpsertRegister(ctx context.Context, tgUserID int64, username, fullName, phone string) error {
	_, err := r.db.ExecContext(ctx, `
		insert into users (tg_user_id, username, full_name, phone)
		values ($1, nullif($2,''), nullif($3,''), nullif($4,''))
		on conflict (tg_user_id)
		do update set
		  username = excluded.username,
		  full_name = excluded.full_name,
		  phone = excluded.phone
	`, tgUserID, username, fullName, phone)
	return err
}
