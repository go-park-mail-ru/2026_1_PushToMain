package app

import (
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/minio"
	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/postgres"

	"github.com/spf13/viper"
)

type AvatarConfig struct {
	MaxSizeMB    int64    `mapstructure:"max_size_mb"`
	AllowedTypes []string `mapstructure:"allowed_types"`
}

type DraftsConfig struct {
	MaxPerUser int `mapstructure:"max_per_user"`
}

type GRPCConfig struct {
	EmailPort string `mapstructure:"email_port"`
}

type GRPCClients struct {
	UserService string `mapstructure:"user_service"`
}

type Config struct {
	ServerPort string `mapstructure:"port"`

	JWTManager utils.JWTManager `mapstructure:"jwt"`

	CORS   middleware.CORSConfig `mapstructure:"cors"`
	Logger logger.Config         `mapstructure:"logger"`

	Db postgres.Config `mapstructure:"postgres"`
	S3 minio.Config    `mapstructure:"minio"`

	Avatar AvatarConfig `mapstructure:"avatar"`
	Drafts DraftsConfig `mapstructure:"drafts"`

	GRPC        GRPCConfig  `mapstructure:"grpc"`
	GRPCClients GRPCClients `mapstructure:"grpc_clients"`
}

func Load(path string) (*Config, error) {

	if err := initConfig(path); err != nil {
		return nil, fmt.Errorf(
			"error initializing config: %w",
			err,
		)
	}

	cfg := &Config{}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf(
			"error unmarshaling config: %w",
			err,
		)
	}

	return cfg, nil
}

func initConfig(path string) error {

	viper.SetConfigType("yaml")

	viper.SetConfigFile(path)

	viper.AutomaticEnv()

	viper.SetEnvKeyReplacer(
		strings.NewReplacer(".", "_"),
	)

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf(
			"error reading config file: %w",
			err,
		)
	}

	fmt.Printf(
		"Using config file: %s\n",
		viper.ConfigFileUsed(),
	)

	return nil
}
