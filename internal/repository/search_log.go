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
	ID            int64
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

func (r *SearchLogsRepo) GetByID(ctx context.Context, logID int64) (*SearchLog, error) {
	var out SearchLog
	err := r.db.QueryRowContext(ctx, `
		select id, tg_user_id, query_type, coalesce(query_text, ''), coalesce(query_image_url, ''), coalesce(result_text, ''), status
		from search_logs
		where id = $1
	`, logID).Scan(&out.ID, &out.TgUserID, &out.QueryType, &out.QueryText, &out.QueryImageURL, &out.ResultText, &out.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get search log: %w", err)
	}
	return &out, nil
}

func (r *SearchLogsRepo) SetStatus(ctx context.Context, logID int64, status string) error {
	_, err := r.db.ExecContext(ctx, `
		update search_logs
		set status = $1, updated_at = now()
		where id = $2
	`, status, logID)
	if err != nil {
		return fmt.Errorf("update search status: %w", err)
	}
	return nil
}
