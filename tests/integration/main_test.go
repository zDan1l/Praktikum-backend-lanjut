//go:build integration

// Package integration_test berisi integration test: test yang berbicara
// dengan PostgreSQL sungguhan. Baris go:build di atas membuat file ini
// HANYA ikut dikompilasi bila perintahnya memakai -tags=integration,
// sehingga "go test ./..." biasa tetap cepat dan tidak butuh database.
package integration_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"pemrograman-code/app/model"
)

// pool dipakai bersama oleh seluruh test di package ini.
var pool *pgxpool.Pool

// TestMain berjalan SEKALI sebelum seluruh test di package ini. Ia
// menyiapkan koneksi dan skema, lalu memanggil m.Run() untuk menjalankan
// test satu per satu.
func TestMain(m *testing.M) {
	// Lokasi relatif terhadap folder package ini: tests/integration.
	_ = godotenv.Load(filepath.Join("..", "..", ".env.test"))

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		fmt.Println("TEST_DATABASE_URL tidak diisi: integration test dilewati")
		os.Exit(0)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		fmt.Println("TEST_DATABASE_URL tidak valid:", err)
		os.Exit(1)
	}

	// PENGAMAN. Integration test menulis dan menghapus data. Test hanya
	// boleh berjalan pada database yang namanya berakhiran _test, agar
	// salah ketik pada .env.test tidak pernah menyentuh database development.
	if !strings.HasSuffix(cfg.ConnConfig.Database, "_test") {
		fmt.Printf("menolak berjalan: database %q tidak berakhiran _test\n", cfg.ConnConfig.Database)
		os.Exit(1)
	}

	// Cukup koneksi agar test konkurensi benar-benar berjalan serentak.
	// Dengan 1 koneksi, sepuluh transaction pendaftaran mengantre satu per
	// satu sehingga tidak pernah bertabrakan, dan hilangnya FOR UPDATE tidak
	// akan pernah terdeteksi. 20 > jumlah request serentak (10).
	cfg.MaxConns = 1

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err == nil {
		err = pool.Ping(ctx)
	}
	if err != nil {
		fmt.Println("gagal terhubung ke database test:", err)
		os.Exit(1)
	}

	if err := applyMigrations(ctx); err != nil {
		fmt.Println("gagal menjalankan migration:", err)
		os.Exit(1)
	}

	// Hangatkan pool: pgxpool membuka koneksi secara malas (saat pertama
	// diminta), dan setiap koneksi baru butuh beberapa milidetik untuk
	// handshake dan autentikasi. Pada pool yang masih dingin, request yang
	// dilepas "serentak" sebenarnya berangkat bergiliran sehingga tabrakan
	// yang dicari test konkurensi kadang tidak terjadi. Seluruh koneksi
	// dibuka di sini SEKALI, sehingga setiap pengulangan — termasuk yang
	// pertama — mendapat koneksi yang sudah siap.
	if err := warmPool(ctx, int(cfg.MaxConns)); err != nil {
		fmt.Println("gagal menyiapkan koneksi:", err)
		os.Exit(1)
	}

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// warmPool membuka n koneksi sekaligus lalu mengembalikannya ke pool sebagai
// koneksi idle yang siap pakai.
func warmPool(ctx context.Context, n int) error {
	conns := make([]*pgxpool.Conn, 0, n)
	defer func() {
		for _, c := range conns {
			c.Release()
		}
	}()
	for range n {
		c, err := pool.Acquire(ctx)
		if err != nil {
			return err
		}
		conns = append(conns, c)
	}
	return nil
}

// applyMigrations menjalankan seluruh file migrations/*.sql berurutan.
// Semua migration kita ditulis dengan IF NOT EXISTS dan ON CONFLICT,
// sehingga aman dijalankan berulang kali.
func applyMigrations(ctx context.Context) error {
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil {
		return err
	}
	slices.Sort(files)
	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

// uniqueSuffix menghasilkan akhiran acak agar data milik setiap test tidak
// pernah bertabrakan dengan data test lain, termasuk saat dijalankan ulang.
func uniqueSuffix(t *testing.T) string {
	t.Helper()
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("membuat akhiran acak: %v", err)
	}
	return hex.EncodeToString(b)
}

// seedStudent membuat satu mahasiswa khusus untuk test yang memanggilnya.
//
// t.Cleanup mendaftarkan pembersihan yang dijalankan SETELAH test selesai
// — berhasil maupun gagal. Seluruh data turunannya (enrollments,
// notifications) ikut terhapus lewat ON DELETE CASCADE.
func seedStudent(t *testing.T) model.User {
	t.Helper()
	ctx := context.Background()
	s := uniqueSuffix(t)

	u := model.User{Username: "it_" + s, Email: "it_" + s + "@student.unair.ac.id", Role: "user"}
	// PENYESUAIAN REPO (bukan kesalahan modul): tabel users di repo ini
	// merujuk role lewat role_id (migration 003), bukan kolom role VARCHAR.
	err := pool.QueryRow(ctx,
		`INSERT INTO users (username, email, password, role_id)
		 VALUES ($1, $2, '!', (SELECT id FROM roles WHERE name = 'viewer')) RETURNING id`,
		u.Username, u.Email).Scan(&u.ID)
	if err != nil {
		t.Fatalf("seed mahasiswa: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM outbox WHERE (payload->>'user_id')::INT = $1`, u.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, u.ID)
	})
	return u
}

// seedCourse membuat satu kelas khusus untuk test yang memanggilnya.
func seedCourse(t *testing.T, credits, quota int) model.Course {
	t.Helper()
	ctx := context.Background()

	c := model.Course{Code: "IT" + uniqueSuffix(t), Name: "Kelas Uji", Credits: credits, Quota: quota}
	err := pool.QueryRow(ctx,
		`INSERT INTO courses (code, name, credits, quota) VALUES ($1, $2, $3, $4) RETURNING id`,
		c.Code, c.Name, c.Credits, c.Quota).Scan(&c.ID)
	if err != nil {
		t.Fatalf("seed kelas: %v", err)
	}

	// t.Cleanup, bukan defer: defer berjalan saat seedCourse SELESAI (kelas
	// langsung terhapus sebelum dipakai), t.Cleanup berjalan saat TEST selesai.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM courses WHERE id = $1`, c.ID)
	})
	return c
}
