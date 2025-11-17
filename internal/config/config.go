package config

import (
	"errors"
	"flag"
	"os"
	"reflect"
	"strings"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const DefaultSecret = "default-secret"

var (
	ErrDatabaseURLRequired = errors.New("database_url is required")
)

type PresetUserInfo struct {
	Role string `yaml:"role"`
}

type Config struct {
	Debug              bool     `yaml:"debug"              envconfig:"DEBUG"`
	Host               string   `yaml:"host"               envconfig:"HOST"`
	Port               string   `yaml:"port"               envconfig:"PORT"`
	BaseURL            string   `yaml:"base_url"          envconfig:"BASE_URL"`
	Secret             string   `yaml:"secret"             envconfig:"SECRET"`
	GoogleClientID     string   `yaml:"google_client_id"   envconfig:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string   `yaml:"google_client_secret"       envconfig:"GOOGLE_CLIENT_SECRET"`
	DatabaseURL        string   `yaml:"database_url"       envconfig:"DATABASE_URL"`
	MigrationSource    string   `yaml:"migration_source"   envconfig:"MIGRATION_SOURCE"`
	AllowOrigins       []string `yaml:"allow_origins"      envconfig:"ALLOW_ORIGINS"`
}

type LogBuffer struct {
	buffer []logEntry
}

type logEntry struct {
	msg  string
	err  error
	meta map[string]string
}

type PresetUserJson struct {
	User string `json:"user"`
	Role string `json:"role"`
}

func NewConfigLogger() *LogBuffer {
	return &LogBuffer{}
}

func (cl *LogBuffer) Warn(msg string, err error, meta map[string]string) {
	cl.buffer = append(cl.buffer, logEntry{msg: msg, err: err, meta: meta})
}

func (cl *LogBuffer) FlushToZap(logger *zap.Logger) {
	for _, e := range cl.buffer {
		var fields []zap.Field
		if e.err != nil {
			fields = append(fields, zap.Error(e.err))
		}
		for k, v := range e.meta {
			fields = append(fields, zap.String(k, v))
		}
		logger.Warn(e.msg, fields...)
	}
	cl.buffer = nil
}

func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return ErrDatabaseURLRequired
	}

	return nil
}

func Load() (Config, *LogBuffer) {
	logger := NewConfigLogger()

	config := &Config{
		Debug:           false,
		Host:            "localhost",
		Port:            "8080",
		Secret:          DefaultSecret,
		DatabaseURL:     "",
		MigrationSource: "file://internal/database/migrations",
	}

	var err error

	config, err = FromFile("config.yaml", config, logger)
	if err != nil {
		logger.Warn("Failed to load config from file", err, map[string]string{"path": "config.yaml"})
	}

	config, err = FromEnv(config, logger)
	if err != nil {
		logger.Warn("Failed to load config from env", err, map[string]string{"path": ".env"})
	}

	config, err = FromFlags(config)
	if err != nil {
		logger.Warn("Failed to load config from flags", err, map[string]string{"path": "flags"})
	}

	return *config, logger
}

func FromFile(filePath string, config *Config, logger *LogBuffer) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return config, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Warn("Failed to close config file", err, map[string]string{"path": filePath})
		}
	}(file)

	fileConfig := Config{}
	if err := yaml.NewDecoder(file).Decode(&fileConfig); err != nil {
		return config, err
	}

	return Merge[Config](config, &fileConfig)
}

func FromEnv(config *Config, logger *LogBuffer) (*Config, error) {
	if err := godotenv.Overload(); err != nil {
		if os.IsNotExist(err) {
			logger.Warn("No .env file found", err, map[string]string{"path": ".env"})
		} else {
			return nil, err
		}
	}

	// Allow origins
	allowOrigins := os.Getenv("ALLOW_ORIGINS")
	if allowOrigins != "" {
		config.AllowOrigins = strings.Split(allowOrigins, ",")
	}

	envConfig := &Config{
		Debug:              os.Getenv("DEBUG") == "true",
		Host:               os.Getenv("HOST"),
		Port:               os.Getenv("PORT"),
		BaseURL:            os.Getenv("BASE_URL"),
		Secret:             os.Getenv("SECRET"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		MigrationSource:    os.Getenv("MIGRATION_SOURCE"),
	}

	return Merge[Config](config, envConfig)
}

func FromFlags(config *Config) (*Config, error) {
	flagConfig := &Config{}

	flag.BoolVar(&flagConfig.Debug, "debug", false, "debug mode")
	flag.StringVar(&flagConfig.Host, "host", "", "host")
	flag.StringVar(&flagConfig.Port, "port", "", "port")
	flag.StringVar(&flagConfig.BaseURL, "base_url", "", "base url")
	flag.StringVar(&flagConfig.Secret, "secret", "", "secret")
	flag.StringVar(&flagConfig.GoogleClientID, "google_client_id", "", "google client id")
	flag.StringVar(&flagConfig.GoogleClientSecret, "google_client_secret", "", "google client secret")
	flag.StringVar(&flagConfig.DatabaseURL, "database_url", "", "database url")
	flag.StringVar(&flagConfig.MigrationSource, "migration_source", "", "migration source")

	flag.Parse()

	return Merge[Config](config, flagConfig)
}

func Merge[T any](base *T, override *T) (*T, error) {
	if base == nil {
		return nil, errors.New("base config cannot be nil")
	}
	if override == nil {
		return base, nil
	}

	final := base
	baseVal := reflect.ValueOf(final).Elem()
	overrideVal := reflect.ValueOf(override).Elem()

	if baseVal.Type() != overrideVal.Type() {
		return nil, errors.New("config types do not match")
	}

	for i := 0; i < baseVal.NumField(); i++ {
		field := baseVal.Field(i)
		overrideField := overrideVal.Field(i)
		zero := reflect.Zero(field.Type()).Interface()

		if field.CanSet() && !reflect.DeepEqual(overrideField.Interface(), zero) {
			if (overrideField.Kind() == reflect.Slice || overrideField.Kind() == reflect.Array) && overrideField.Len() == 0 {
				continue
			}
			field.Set(overrideField)
		}
	}

	return final, nil
}
