package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name        string
		config      *LoggerConfig
		expectError bool
	}{
		{
			name: "valid development config",
			config: &LoggerConfig{
				Level:           "debug",
				Environment:     "development",
				OutputPath:      "stdout",
				ErrorOutputPath: "stderr",
			},
			expectError: false,
		},
		{
			name: "valid production config",
			config: &LoggerConfig{
				Level:           "info",
				Environment:     "production",
				OutputPath:      "stdout",
				ErrorOutputPath: "stderr",
			},
			expectError: false,
		},
		{
			name: "invalid log level",
			config: &LoggerConfig{
				Level:           "invalid",
				Environment:     "development",
				OutputPath:      "stdout",
				ErrorOutputPath: "stderr",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(tt.config)
			if (err != nil) != tt.expectError {
				t.Errorf("Init() error = %v, expectError %v", err, tt.expectError)
			}

			if Logger == nil && !tt.expectError {
				t.Error("Log is nil after successful initialization")
			}

			Sync()
		})
	}
}

func TestLoggerInitialization(t *testing.T) {
	cfg := &LoggerConfig{
		Level:           "info",
		Environment:     "test",
		OutputPath:      "stdout",
		ErrorOutputPath: "stderr",
	}

	err := Init(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Logger.Info("test message")

	assert.NotPanics(t, func() {
		Logger.Infow("structured test",
			"key", "value",
			"number", 42,
		)
	})

	assert.NotPanics(t, func() {
		Sync()
	})
}
