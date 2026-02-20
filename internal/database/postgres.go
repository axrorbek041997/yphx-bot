package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"yphx-bot/internal/config" // <- o'zingning module nomingga mosla
)

func ConnectPostgres(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Pool sozlamalari
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdle)
	db.SetConnMaxLifetime(cfg.ConnMaxLife)

	// Startup ping (timeout bilan)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.PingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	return db, nil
}
