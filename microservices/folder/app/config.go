package app

import (
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/postgres"

	"github.com/spf13/viper"
)

type GRPCConfig struct {
	FolderPort string `mapstructure:"folder_port"`
}

type GRPCClients struct {
	EmailService string `mapstructure:"email_service"`
}

type Config struct {
	ServerPort string `mapstructure:"port"`

	JWTManager utils.JWTManager `mapstructure:"jwt"`

	CORS   middleware.CORSConfig `mapstructure:"cors"`
	Logger logger.Config         `mapstructure:"logger"`

	Db postgres.Config `mapstructure:"postgres"`

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
