package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type AppConfig struct {
	Port             string `mapstructure:"PORT"`
	PostgresUsername string `mapstructure:"POSTGRES_USERNAME"`
	PostgresPassword string `mapstructure:"POSTGRES_PASSWORD"`
	PostgresDatabase string `mapstructure:"POSTGRES_DATABASE"`
	PostgresSSLMode  string `mapstructure:"POSTGRES_SSLMODE"`
	PostgresHost     string `mapstructure:"POSTGRES_HOST"`
	PostgresPort     string `mapstructure:"POSTGRES_PORT"`
	RabbitMQURL      string `mapstructure:"RABBITMQ_URL"`
	ServiceName      string `mapstructure:"SERVICE_NAME"`
}

func Read() *AppConfig {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	_ = viper.ReadInConfig()

	viper.AutomaticEnv()

	bindEnvVariables()
	setDefaults()

	var appConfig AppConfig
	err := viper.Unmarshal(&appConfig)
	if err != nil {
		panic(fmt.Errorf("fatal error unmarshalling config: %w", err))
	}

	return &appConfig
}

func bindEnvVariables() {
	_ = viper.BindEnv("PORT")
	_ = viper.BindEnv("POSTGRES_USERNAME")
	_ = viper.BindEnv("POSTGRES_PASSWORD")
	_ = viper.BindEnv("POSTGRES_DATABASE")
	_ = viper.BindEnv("POSTGRES_SSLMODE")
	_ = viper.BindEnv("POSTGRES_HOST")
	_ = viper.BindEnv("POSTGRES_PORT")
	_ = viper.BindEnv("RABBITMQ_URL")
	_ = viper.BindEnv("SERVICE_NAME")
}

func setDefaults() {
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("POSTGRES_SSLMODE", "disable")
	viper.SetDefault("POSTGRES_HOST", "localhost")
	viper.SetDefault("POSTGRES_PORT", "5432")
	viper.SetDefault("SERVICE_NAME", "auction")
}
