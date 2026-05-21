package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type Config struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`

	MigrationUser     string `mapstructure:"migration_user"`
	MigrationPassword string `mapstructure:"migration_password"`
	MigrationsPath    string `mapstructure:"migrations_path"`

	Pool PoolConfig `mapstructure:"pool"`
}

type PoolConfig struct {
	MaxOpenConns       int `mapstructure:"max_open_conns"`
	MaxIdleConns       int `mapstructure:"max_idle_conns"`
	ConnMaxLifetimeSec int `mapstructure:"conn_max_lifetime_sec"`
	ConnMaxIdleTimeSec int `mapstructure:"conn_max_idle_time_sec"`
}

func (cfg *Config) ToDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)
}

func (cfg *Config) toMigrationDSN() string {
	user, password := cfg.MigrationUser, cfg.MigrationPassword
	if user == "" {
		user, password = cfg.User, cfg.Password
	}

	return fmt.Sprintf("pgx://%s:%s@%s:%d/%s?sslmode=%s", user, password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)
}

func Ping(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}

	return nil
}

type Opener func(driverName, dataSourceName string) (*sql.DB, error)

func NewWithOpener(cfg Config, opener Opener) (*sql.DB, error) {
	dsn := cfg.ToDSN()
	return opener("pgx", dsn)
}

func New(cfg Config) (*sql.DB, error) {
	db, err := NewWithOpener(cfg, sql.Open)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.Pool.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Pool.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.Pool.ConnMaxLifetimeSec) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(cfg.Pool.ConnMaxIdleTimeSec) * time.Second)

	return db, nil
}

func RunMigrations(cfg Config) error {
	dsn := cfg.toMigrationDSN()

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
