package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Этот импорт критически важен!
)

const (
	migrationsPath = "file://db/migrations"
)

type Config struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

func (cfg *Config) ToDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)
}

func Ping(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}

	return nil
}

func New(cfg Config) (*sql.DB, error) {
	dsn := cfg.ToDSN()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func RunMigrations(cfg Config) error {
	dsn := cfg.ToDSN()

	m, err := migrate.New(migrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("cannot create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("cannot apply migrations: %v", err)
	}

	if err == migrate.ErrNoChange {
		return fmt.Errorf("✅ No new migrations to apply")
	} else {
		fmt.Println("Migrations applied successfully")
	}

	return nil
}
