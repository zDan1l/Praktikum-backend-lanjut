package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"pemrograman-code/app/model"
)

type CommentRepository struct {
	pool *pgxpool.Pool
}

func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{pool: pool}
}

func (r *CommentRepository) ArticleExists(ctx context.Context, articleID int) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM articles WHERE id = $1)`, articleID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("mengecek artikel: %w", err)
	}
	return exists, nil
}

func (r *CommentRepository) FindByID(ctx context.Context, id int) (model.Comment, error) {
	var c model.Comment
	err := r.pool.QueryRow(ctx,
		`SELECT id, article_id, author_id, content, created_at
		 FROM comments WHERE id = $1`, id,
	).Scan(&c.ID, &c.ArticleID, &c.AuthorID, &c.Content, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Comment{}, ErrNotFound
	}
	if err != nil {
		return model.Comment{}, fmt.Errorf("mengambil komentar: %w", err)
	}
	return c, nil
}

func (r *CommentRepository) ListByArticle(ctx context.Context, articleID int) ([]model.Comment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, article_id, author_id, content, created_at
		 FROM comments WHERE article_id = $1 ORDER BY id ASC`, articleID,
	)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar komentar: %w", err)
	}
	defer rows.Close()

	comments := []model.Comment{}
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.ID, &c.ArticleID, &c.AuthorID, &c.Content, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("membaca baris komentar: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (r *CommentRepository) Create(ctx context.Context, c model.Comment) (model.Comment, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO comments (article_id, author_id, content)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		c.ArticleID, c.AuthorID, c.Content,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return model.Comment{}, fmt.Errorf("menyimpan komentar: %w", err)
	}
	return c, nil
}

func (r *CommentRepository) UpdateContent(ctx context.Context, id int, content string) (model.Comment, error) {
	var c model.Comment
	err := r.pool.QueryRow(ctx,
		`UPDATE comments SET content = $1 WHERE id = $2
		 RETURNING id, article_id, author_id, content, created_at`,
		content, id,
	).Scan(&c.ID, &c.ArticleID, &c.AuthorID, &c.Content, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Comment{}, ErrNotFound
	}
	if err != nil {
		return model.Comment{}, fmt.Errorf("memperbarui komentar: %w", err)
	}
	return c, nil
}

func (r *CommentRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM comments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("menghapus komentar: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
