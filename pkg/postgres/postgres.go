package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type Config struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	User           string `mapstructure:"user"`
	Password       string `mapstructure:"password"`
	DBName         string `mapstructure:"dbname"`
	SSLMode        string `mapstructure:"sslmode"`
	MigrationsPath string `mapstructure:"migrations_path"`
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

	m, err := migrate.New(cfg.MigrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("cannot create migrate instance: %w", err)
	}
	defer m.Close()

	errUp := m.Up()
	if errUp != nil && !errors.Is(errUp, migrate.ErrNoChange) {
		return fmt.Errorf("cannot apply migrations: %w", errUp)
	}

	if errors.Is(errUp, migrate.ErrNoChange) {
		fmt.Println("No new migrations to apply")
	} else {
		fmt.Printf("Migrations applied successfully from")
	}

	return nil
}
