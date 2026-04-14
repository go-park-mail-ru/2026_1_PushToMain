package postgres

import "testing"

func TestConfig_ToDSN(t *testing.T) {
	cfg := &Config{
		Host:     "localhost",
		Port:     5034,
		User:     "admin",
		Password: "12345",
		DBName:   "pqdb",
		SSLMode:  "disable",
	}

	expected := "postgres://admin:12345@localhost:5034/pqdb?sslmode=disable"
	actual := cfg.ToDSN()
	if expected != actual {
		t.Errorf("got %s, expected %s", actual, expected)
	}
}
