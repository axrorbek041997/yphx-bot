package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

type VectorsRepo struct {
	db *sql.DB
}

func NewVectorsRepo(db *sql.DB) *VectorsRepo {
	return &VectorsRepo{db: db}
}

func (r *VectorsRepo) Save(ctx context.Context, typ, text, imageURL string, vector []float64) error {
	vectorLiteral := toPgVectorLiteral(vector)
	_, err := r.db.ExecContext(ctx, `
		insert into vectors (type, text, image_url, vector)
		values ($1, nullif($2, ''), nullif($3, ''), $4::vector)
	`, typ, text, imageURL, vectorLiteral)
	if err != nil {
		return fmt.Errorf("insert vector: %w", err)
	}

	return nil
}

func toPgVectorLiteral(vector []float64) string {
	parts := make([]string, 0, len(vector))
	for _, v := range vector {
		parts = append(parts, strconv.FormatFloat(v, 'f', -1, 64))
	}
	return "[" + strings.Join(parts, ",") + "]"
}
