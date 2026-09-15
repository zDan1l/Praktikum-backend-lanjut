package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// common.go - helper kecil biar repository tidak duplikat
// Cara tambah repository baru: copy student_repository.go / prestasi_repository.go,
// ganti nama tabel, daftar kolom, dan fungsi buildFilter.

func orderDir(order string) string {
	if order == "desc" {
		return "DESC"
	}
	return "ASC"
}

func sortCol(sort string, whitelist map[string]string) string {
	if col, ok := whitelist[sort]; ok && col != "" {
		return col
	}
	return "id"
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	return false
}
