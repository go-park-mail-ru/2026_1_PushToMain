package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Level           string `mapstructure:"level"`
	Environment     string `mapstructure:"environment"`
	OutputPath      string `mapstructure:"output_path"`
	ErrorOutputPath string `mapstructure:"error_output_path"`
}

func New(cfg *Config) (*zap.SugaredLogger, error) {
	var zapConfig zap.Config

	if cfg.Environment == "development" {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	zapConfig.OutputPaths = []string{cfg.OutputPath}
	zapConfig.ErrorOutputPaths = []string{cfg.ErrorOutputPath}

	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	zapLogger, err := zapConfig.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}

	Logger := zapLogger.Sugar()

	return Logger, nil
}

func Sync(logger *zap.SugaredLogger) error {
	if logger != nil {
		return logger.Sync()
	}
	return nil
}
