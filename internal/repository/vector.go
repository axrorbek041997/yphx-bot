package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

type VectorSearchResult struct {
	ID       int64
	Text     sql.NullString
	ImageURL sql.NullString
	Type     string
	Score    float64
}

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

func (r *VectorsRepo) SearchSimilar(ctx context.Context, typ string, queryVector []float64, limit int) ([]VectorSearchResult, error) {
	vectorLiteral := toPgVectorLiteral(queryVector)
	rows, err := r.db.QueryContext(ctx, `
		select id, text, image_url, type, 1 - (vector <=> $1::vector) as score
		from vectors
		where type = $2
		order by vector <=> $1::vector
		limit $3
	`, vectorLiteral, typ, limit)
	if err != nil {
		return nil, fmt.Errorf("search vectors: %w", err)
	}
	defer rows.Close()

	out := make([]VectorSearchResult, 0, limit)
	for rows.Next() {
		var row VectorSearchResult
		if err := rows.Scan(&row.ID, &row.Text, &row.ImageURL, &row.Type, &row.Score); err != nil {
			return nil, fmt.Errorf("scan vectors: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vectors: %w", err)
	}

	return out, nil
}

func toPgVectorLiteral(vector []float64) string {
	parts := make([]string, 0, len(vector))
	for _, v := range vector {
		parts = append(parts, strconv.FormatFloat(v, 'f', -1, 64))
	}
	return "[" + strings.Join(parts, ",") + "]"
}
