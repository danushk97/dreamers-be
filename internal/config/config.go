package config

import (
	"os"

	"github.com/spf13/viper"
)

// Config holds application configuration.
type Config struct {
	Server            ServerConfig
	Database          DatabaseConfig
	GDrive            GDriveConfig
	GDriveCredentials []byte `mapstructure:"-"` // loaded at runtime
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL           string
	MigrationPath string
}

type GDriveConfig struct {
	CredentialsPath string
	FolderID        string
	MaxSizeMB       int64
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
	v.SetDefault("gdrive.max_size_mb", 2)

	v.SetEnvPrefix("")
	v.AutomaticEnv()
	_ = v.BindEnv("server.port", "PORT")
	_ = v.BindEnv("database.url", "DATABASE_URL")
	_ = v.BindEnv("database.migration_path", "MIGRATION_PATH")
	_ = v.BindEnv("gdrive.folder_id", "GDRIVE_FOLDER_ID")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	if path := v.GetString("gdrive.credentials_path"); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			cfg.GDriveCredentials = data
		}
	}
	if path := os.Getenv("GDRIVE_CREDENTIALS_JSON"); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			cfg.GDriveCredentials = data
		}
	}
	if s := os.Getenv("GDRIVE_CREDENTIALS"); s != "" {
		cfg.GDriveCredentials = []byte(s)
	}

	return cfg, nil
}
