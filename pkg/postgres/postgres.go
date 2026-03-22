package postgres

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func (cfg *Config) ToDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}

func Ping(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}

	return nil
}

func New(ctx context.Context, cfg Config) (*sql.DB, error) {
	dsn := cfg.ToDSN()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return db, err
	}

	return db, nil
}
