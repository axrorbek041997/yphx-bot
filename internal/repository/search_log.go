package repository

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	SearchStatusSuccess  = "success"
	SearchStatusNotFound = "not_found"
	SearchStatusIgnored  = "ignored"
)

type SearchLog struct {
	TgUserID      int64
	QueryType     string
	QueryText     string
	QueryImageURL string
	ResultText    string
	Status        string
}

type SearchLogsRepo struct {
	db *sql.DB
}

func NewSearchLogsRepo(db *sql.DB) *SearchLogsRepo { return &SearchLogsRepo{db: db} }

func (r *SearchLogsRepo) Create(ctx context.Context, log SearchLog) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		insert into search_logs (tg_user_id, query_type, query_text, query_image_url, result_text, status)
		values ($1, $2, nullif($3, ''), nullif($4, ''), nullif($5, ''), $6)
		returning id
	`, log.TgUserID, log.QueryType, log.QueryText, log.QueryImageURL, log.ResultText, log.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create search log: %w", err)
	}
	return id, nil
}

func (r *SearchLogsRepo) SetReaction(ctx context.Context, logID, tgUserID int64, reaction string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		update search_logs
		set reaction = $1, updated_at = now()
		where id = $2 and tg_user_id = $3 and status = 'success'
	`, reaction, logID, tgUserID)
	if err != nil {
		return false, fmt.Errorf("set reaction: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("reaction rows affected: %w", err)
	}
	return affected > 0, nil
}
