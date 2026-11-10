package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"pemrograman-code/config"
)

func NewMongo(ctx context.Context) (*mongo.Client, *mongo.Database, error) {
	timeout := time.Duration(config.GetEnvInt("MONGO_TIMEOUT_SECONDS", 10)) * time.Second

	client, err := mongo.Connect(options.Client().
		ApplyURI(config.GetEnv("MONGO_URI", "mongodb://localhost:27017")).
		SetServerSelectionTimeout(timeout))
	if err != nil {
		return nil, nil, fmt.Errorf("konfigurasi MongoDB tidak valid: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, fmt.Errorf("gagal terhubung ke MongoDB: %w", err)
	}

	// MONGO_DATABASE dipakai docker-compose (Modul 12); MONGO_DB adalah nama
	// lama di .env sejak Modul 7. Keduanya diterima.
	name := config.GetEnv("MONGO_DATABASE", config.GetEnv("MONGO_DB", "praktikum_backend"))
	db := client.Database(name)
	return client, db, nil
}
