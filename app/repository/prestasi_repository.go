package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"pemrograman-code/app/model"
)

type PrestasiRepository interface {
	FindAll(ctx context.Context, q model.ListQuery) ([]model.Prestasi, int, error)
	FindByID(ctx context.Context, id int) (model.Prestasi, error)
}

var kolomPrestasi = map[string]string{
	"id":            "id",
	"id_student":    "id_student",
	"nama_prestasi": "nama_prestasi",
	"created_at":    "created_at",
	"juara":         "juara",
}

type prestasiPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPrestasiRepository(pool *pgxpool.Pool) PrestasiRepository {
	return &prestasiPostgresRepository{pool: pool}
}

func buildPrestasiFilter(q model.ListQuery) (string, []any) {
	where := " WHERE 1 = 1"
	args := []any{}
	if q.Search != "" {
		where += fmt.Sprintf(" AND nama_prestasi ILIKE $%d", len(args)+1)
		args = append(args, "%"+q.Search+"%")
	}
	// is_active tidak ada di tabel prestasi, jadi diabaikan agar tidak error "column does not exist"
	return where, args
}

func (r *prestasiPostgresRepository) FindAll(ctx context.Context, q model.ListQuery) ([]model.Prestasi, int, error) {
	where, args := buildPrestasiFilter(q)
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM prestasi"+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("menghitung prestasi: %w", err)
	}
	arah := "ASC"
	if q.Order == "desc" {
		arah = "DESC"
	}
	kolom := kolomPrestasi[q.Sort]
	if kolom == "" {
		kolom = "id"
	}
	sqlText := fmt.Sprintf(
		`SELECT id, id_student, nama_prestasi, created_at, juara
	 FROM prestasi%s
	 ORDER BY %s %s
	 LIMIT $%d OFFSET $%d`,
		where, kolom, arah, len(args)+1, len(args)+2,
	)
	args = append(args, q.Limit, q.Offset())
	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("mengambil daftar prestasi: %w", err)
	}
	defer rows.Close()
	hasil := []model.Prestasi{}
	for rows.Next() {
		var s model.Prestasi
		if err := rows.Scan(&s.ID, &s.ID_Student, &s.NamaPrestasi, &s.CreatedAt, &s.Juara); err != nil {
			return nil, 0, fmt.Errorf("membaca baris prestasi: %w", err)
		}
		hasil = append(hasil, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("membaca hasil query: %w", err)
	}
	return hasil, total, nil
}

func (r *prestasiPostgresRepository) FindByID(ctx context.Context, id int) (model.Prestasi, error) {
	var s model.Prestasi
	err := r.pool.QueryRow(ctx,
		`SELECT id, id_student, nama_prestasi, created_at, juara
	 FROM prestasi WHERE id = $1`, id,
	).Scan(&s.ID, &s.ID_Student, &s.NamaPrestasi, &s.CreatedAt, &s.Juara)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Prestasi{}, ErrNotFound
		}
		return model.Prestasi{}, fmt.Errorf("mengambil prestasi: %w", err)
	}
	return s, nil
}
