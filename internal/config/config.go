package config

import (
	"github.com/spf13/viper"
)

// Config holds application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	S3       S3Config
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL           string
	MigrationPath string
}

type S3Config struct {
	Bucket    string `mapstructure:"bucket"`
	Region    string `mapstructure:"region"`
	BaseURL   string `mapstructure:"base_url"`
	MaxSizeMB int64  `mapstructure:"max_size_mb"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
}

// Load reads configuration from config.toml and environment.
func Load() (Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("..")

	v.SetDefault("server.port", "8080")
	v.SetDefault("database.url", "postgres://postgres:postgres@localhost:5432/dreamers?sslmode=disable")
	v.SetDefault("database.migration_path", "./migrations")
	v.SetDefault("s3.region", "us-east-1")
	v.SetDefault("s3.max_size_mb", 2)

	v.SetEnvPrefix("")
	v.AutomaticEnv()
	_ = v.BindEnv("server.port", "PORT")
	_ = v.BindEnv("database.url", "DATABASE_URL")
	_ = v.BindEnv("database.migration_path", "MIGRATION_PATH")
	_ = v.BindEnv("s3.bucket", "AWS_S3_BUCKET")
	_ = v.BindEnv("s3.region", "AWS_REGION")
	_ = v.BindEnv("s3.access_key", "AWS_ACCESS_KEY_ID")
	_ = v.BindEnv("s3.secret_key", "AWS_SECRET_ACCESS_KEY")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
