package postgres

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_ToDSN(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		DBName:   "db",
		SSLMode:  "disable",
	}

	dsn := cfg.ToDSN()

	expected := "postgres://user:pass@localhost:5432/db?sslmode=disable"
	if dsn != expected {
		t.Fatalf("expected %s, got %s", expected, dsn)
	}
}

func TestConfig_ToDSNPGX(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		DBName:   "db",
		SSLMode:  "disable",
	}

	dsn := cfg.ToDSNPGX()

	expected := "pgx://user:pass@localhost:5432/db?sslmode=disable"
	if dsn != expected {
		t.Fatalf("expected %s, got %s", expected, dsn)
	}
}

func TestPing(t *testing.T) {
	tests := []struct {
		name        string
		mockSetup   func(mock sqlmock.Sqlmock)
		expectedErr error
	}{
		{
			name: "successful ping",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing()
			},
			expectedErr: nil,
		},
		{
			name: "ping fails",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectPing().WillReturnError(errors.New("connection refused"))
			},
			expectedErr: errors.New("postgres ping: connection refused"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)
			defer db.Close()

			tt.mockSetup(mock)

			err = Ping(db)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestNewWithOpener(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		DBName:   "test",
		SSLMode:  "disable",
	}

	t.Run("success", func(t *testing.T) {
		mockDB, _, _ := sqlmock.New()
		defer mockDB.Close()

		opener := func(driverName, dsn string) (*sql.DB, error) {
			assert.Equal(t, "pgx", driverName)
			assert.Equal(t, cfg.ToDSN(), dsn)
			return mockDB, nil
		}

		db, err := NewWithOpener(cfg, opener)
		assert.NoError(t, err)
		assert.Equal(t, mockDB, db)
	})

	t.Run("opener returns error", func(t *testing.T) {
		opener := func(driverName, dsn string) (*sql.DB, error) {
			return nil, errors.New("cannot open database")
		}

		db, err := NewWithOpener(cfg, opener)
		assert.Nil(t, db)
		assert.EqualError(t, err, "cannot open database")
	})
}

func TestNew(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		DBName:   "db",
		SSLMode:  "disable",
	}

	db, err := New(cfg)

	// sql.Open НЕ проверяет соединение → ошибки не будет
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if db == nil {
		t.Fatal("expected db, got nil")
	}
}
