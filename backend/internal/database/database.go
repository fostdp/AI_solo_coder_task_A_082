package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"grotto-monitor/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func Init(cfg config.DatabaseConfig) error {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("parse database config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	DB, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("create connection pool: %w", err)
	}

	if err = DB.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	log.Println("✓ TimescaleDB database connection established")
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
		log.Println("✓ Database connection pool closed")
	}
}

func GetPool() *pgxpool.Pool {
	return DB
}
