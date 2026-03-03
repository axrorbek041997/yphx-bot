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

func (r *VectorsRepo) SaveText(ctx context.Context, text string, textVector []float64) error {
	vectorLiteral := toPgVectorLiteral(textVector)
	_, err := r.db.ExecContext(ctx, `
		insert into vectors (type, text, text_vector, vector)
		values ('text', nullif($1, ''), $2::vector, $2::vector)
	`, text, vectorLiteral)
	if err != nil {
		return fmt.Errorf("insert text vector: %w", err)
	}
	return nil
}

func (r *VectorsRepo) SaveImage(ctx context.Context, text, imageURL string, imageVector []float64, textVector []float64) error {
	imageVectorLiteral := toPgVectorLiteral(imageVector)
	var textVectorArg any
	if len(textVector) > 0 {
		textVectorArg = toPgVectorLiteral(textVector)
	}

	_, err := r.db.ExecContext(ctx, `
		insert into vectors (type, text, image_url, image_vector, text_vector, vector)
		values ('image', nullif($1, ''), nullif($2, ''), $3::vector, $4::vector, $3::vector)
	`, text, imageURL, imageVectorLiteral, textVectorArg)
	if err != nil {
		return fmt.Errorf("insert image vector: %w", err)
	}
	return nil
}

func (r *VectorsRepo) SearchSimilarText(ctx context.Context, queryVector []float64, limit int) ([]VectorSearchResult, error) {
	vectorLiteral := toPgVectorLiteral(queryVector)
	rows, err := r.db.QueryContext(ctx, `
		select id, text, image_url, type, 1 - (text_vector <=> $1::vector) as score
		from vectors
		where type = 'text' and text_vector is not null
		order by text_vector <=> $1::vector
		limit $2
	`, vectorLiteral, limit)
	if err != nil {
		return nil, fmt.Errorf("search text vectors: %w", err)
	}
	defer rows.Close()

	out := make([]VectorSearchResult, 0, limit)
	for rows.Next() {
		var row VectorSearchResult
		if err := rows.Scan(&row.ID, &row.Text, &row.ImageURL, &row.Type, &row.Score); err != nil {
			return nil, fmt.Errorf("scan text vectors: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate text vectors: %w", err)
	}

	return out, nil
}

func (r *VectorsRepo) SearchSimilarImage(ctx context.Context, queryVector []float64, limit int) ([]VectorSearchResult, error) {
	vectorLiteral := toPgVectorLiteral(queryVector)
	rows, err := r.db.QueryContext(ctx, `
		select id, text, image_url, type, 1 - (image_vector <=> $1::vector) as score
		from vectors
		where type = 'image' and image_vector is not null
		order by image_vector <=> $1::vector
		limit $2
	`, vectorLiteral, limit)
	if err != nil {
		return nil, fmt.Errorf("search image vectors: %w", err)
	}
	defer rows.Close()

	out := make([]VectorSearchResult, 0, limit)
	for rows.Next() {
		var row VectorSearchResult
		if err := rows.Scan(&row.ID, &row.Text, &row.ImageURL, &row.Type, &row.Score); err != nil {
			return nil, fmt.Errorf("scan image vectors: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate image vectors: %w", err)
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
