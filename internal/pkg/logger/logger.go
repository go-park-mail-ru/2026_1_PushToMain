package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.SugaredLogger

type LoggerConfig struct {
	Level           string `mapstructure:"level"`
	Environment     string `mapstructure:"environment"`
	OutputPath      string `mapstructure:"output_path"`
	ErrorOutputPath string `mapstructure:"error_output_path"`
}

func Init(cfg *LoggerConfig) error {
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
		return err
	}

	Logger = zapLogger.Sugar()

	return nil
}

func DefaultConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:           "info",
		Environment:     "development",
		OutputPath:      "stdout",
		ErrorOutputPath: "stderr",
	}
}

func Sync() error {
	if Logger != nil {
		return Logger.Sync()
	}
	return nil
}
