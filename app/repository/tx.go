package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX adalah apa saja yang dapat menjalankan query: baik *pgxpool.Pool
// (di luar transaction) maupun pgx.Tx (di dalam transaction). Repository
// menerima DBTX supaya method yang sama bisa berjalan di dalam atau di luar
// transaction tanpa ditulis dua kali.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// TxManager menjalankan sekumpulan operasi di dalam satu transaction
// PostgreSQL.
type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager { return &TxManager{pool: pool} }

const maxTxRetry = 3

// WithTx menjalankan fn di dalam satu transaction. Bila transaction gagal
// karena deadlock (40P01) atau serialization failure (40001), SELURUH fn
// diulang dari awal, paling banyak maxTxRetry kali.
//
// PERINGATAN: karena fn bisa dijalankan lebih dari sekali, jangan melakukan
// efek samping di luar database di dalamnya (mengirim email, enqueue task) —
// efek itu tidak ikut di-rollback dan akan terulang. Untuk kasus seperti itu
// pakai WithTxOnce.
func (m *TxManager) WithTx(ctx context.Context, fn func(tx DBTX) error) error {
	var err error
	for attempt := 0; attempt < maxTxRetry; attempt++ {
		err = m.runOnce(ctx, fn)
		if !isRetryable(err) {
			return err
		}
	}
	return err
}

// WithTxOnce sama dengan WithTx, tetapi TIDAK PERNAH melakukan retry.
//
// Dipakai ketika fn memang harus melakukan sesuatu di luar database —
// mengirim email, melakukan enqueue task — sambil memegang lock di database.
// Melakukan retry fn berarti mengulang tindakan luar itu. Tanggung jawab
// retry dipindahkan ke pemanggil: worker queue sudah punya mekanisme retry
// sendiri, dan relay outbox akan mencoba lagi pada putaran berikutnya.
func (m *TxManager) WithTxOnce(ctx context.Context, fn func(tx DBTX) error) error {
	return m.runOnce(ctx, fn)
}

func (m *TxManager) runOnce(ctx context.Context, fn func(tx DBTX) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("memulai transaction: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func isRetryable(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40P01" || pgErr.Code == "40001"
	}
	return false
}
