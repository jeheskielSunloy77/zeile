package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	_ "github.com/joho/godotenv/autoload"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
)

type Config struct {
	Primary       Primary              `koanf:"primary" validate:"required"`
	Server        ServerConfig         `koanf:"server" validate:"required"`
	Database      DatabaseConfig       `koanf:"database" validate:"required"`
	Auth          AuthConfig           `koanf:"auth" validate:"required"`
	Cache         CacheConfig          `koanf:"cache" validate:"required"`
	FileStorage   FileStorageConfig    `koanf:"file_storage"`
	SMTP          SMTPConfig           `koanf:"smtp" validate:"required"`
	Observability *ObservabilityConfig `koanf:"observability"`
	Seeder        SeederConfig         `koanf:"seeder" validate:"required"`
}

type Env string

const (
	EnvDevelopment Env = "development"
	EnvStaging     Env = "staging"
	EnvProduction  Env = "production"
)

type Primary struct {
	Env     Env    `koanf:"env" validate:"required,oneof=development staging production"`
	AppName string `koanf:"app_name" validate:"required"`
}

type ServerConfig struct {
	Port               string        `koanf:"port" validate:"required"`
	ReadTimeout        time.Duration `koanf:"read_timeout" validate:"required"`
	WriteTimeout       time.Duration `koanf:"write_timeout" validate:"required"`
	IdleTimeout        time.Duration `koanf:"idle_timeout" validate:"required"`
	CORSAllowedOrigins []string      `koanf:"cors_allowed_origins" validate:"required"`
}

type SSLMode string

const (
	SSLModeDisable SSLMode = "disable"
	SSLModeRequire SSLMode = "require"
	SSLModeVerify  SSLMode = "verify-full"
)

type DatabaseConfig struct {
	Host            string        `koanf:"host" validate:"required"`
	Port            int           `koanf:"port" validate:"required"`
	User            string        `koanf:"user" validate:"required"`
	Password        string        `koanf:"password"`
	Name            string        `koanf:"name" validate:"required"`
	SSLMode         SSLMode       `koanf:"ssl_mode" validate:"required,oneof=disable require verify-full"`
	MaxOpenConns    int           `koanf:"max_open_conns" validate:"required"`
	MaxIdleConns    int           `koanf:"max_idle_conns" validate:"required"`
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime" validate:"required"`
	ConnMaxIdleTime time.Duration `koanf:"conn_max_idle_time" validate:"required"`
}

type SeederConfig struct {
	Enabled bool          `koanf:"enabled" validate:"required"`
	Timeout time.Duration `koanf:"timeout" validate:"required"`
}

type CacheConfig struct {
	TTL          time.Duration `koanf:"ttl" validate:"required"`
	RedisAddress string        `koanf:"redis_address" validate:"required"`
}

type FileStorageConfig struct {
	Provider         string                 `koanf:"provider"`
	MaxUploadSizeMB  int                    `koanf:"max_upload_size_mb"`
	AllowedMimeTypes []string               `koanf:"allowed_mime_types"`
	PathPrefix       string                 `koanf:"path_prefix"`
	Local            FileStorageLocalConfig `koanf:"local"`
	S3               FileStorageS3Config    `koanf:"s3"`
}

type FileStorageLocalConfig struct {
	BaseDir    string `koanf:"base_dir"`
	PublicPath string `koanf:"public_path"`
}

type FileStorageS3Config struct {
	Bucket          string `koanf:"bucket"`
	Region          string `koanf:"region"`
	Endpoint        string `koanf:"endpoint"`
	PublicURL       string `koanf:"public_url"`
	AccessKeyID     string `koanf:"access_key_id"`
	SecretAccessKey string `koanf:"secret_access_key"`
	ForcePathStyle  bool   `koanf:"force_path_style"`
}

type IntegrationConfig struct {
	SMTP SMTPConfig `koanf:"smtp" validate:"required"`
}

type SMTPConfig struct {
	Host      string `koanf:"host" validate:"required"`
	Port      int    `koanf:"port" validate:"required"`
	Username  string `koanf:"username" validate:"required"`
	Password  string `koanf:"password" validate:"required"`
	FromEmail string `koanf:"from_email" validate:"required,email"`
	FromName  string `koanf:"from_name" validate:"required"`
}

type CookieSameSite string

const (
	CookieSameSiteLax    CookieSameSite = "lax"
	CookieSameSiteStrict CookieSameSite = "strict"
	CookieSameSiteNone   CookieSameSite = "none"
)

type AuthConfig struct {
	SecretKey                string         `koanf:"secret_key" validate:"required"`
	AccessTokenTTL           time.Duration  `koanf:"access_token_ttl" validate:"required"`
	RefreshTokenTTL          time.Duration  `koanf:"refresh_token_ttl" validate:"required"`
	GoogleClientID           string         `koanf:"google_client_id"`
	GoogleClientSecret       string         `koanf:"google_client_secret"`
	GoogleRedirectURL        string         `koanf:"google_redirect_url"`
	GoogleSuccessRedirectURL string         `koanf:"google_success_redirect_url"`
	GoogleFailureRedirectURL string         `koanf:"google_failure_redirect_url"`
	EmailVerificationTTL     time.Duration  `koanf:"email_verification_ttl" validate:"required"`
	AccessCookieName         string         `koanf:"access_cookie_name" validate:"required"`
	RefreshCookieName        string         `koanf:"refresh_cookie_name" validate:"required"`
	CookieDomain             string         `koanf:"cookie_domain"`
	CookieSameSite           CookieSameSite `koanf:"cookie_same_site" validate:"required,oneof=lax strict none"`
}

func LoadConfig() (*Config, error) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	k := koanf.New(".")

	err := k.Load(env.Provider("API_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "API_"))
	}), nil)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not load initial env variables")
	}

	mainConfig := &Config{}

	err = k.Unmarshal("", mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not unmarshal main config")
	}

	validate := validator.New()

	err = validate.Struct(mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("config validation failed")
	}

	if err := validateFileStorageConfig(mainConfig); err != nil {
		logger.Fatal().Err(err).Msg("file storage config validation failed")
	}

	// Set default observability config if not provided
	if mainConfig.Observability == nil {
		mainConfig.Observability = DefaultObservabilityConfig()
	}

	// Validate observability config
	if err := mainConfig.Observability.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("invalid observability config")
	}

	// If Google Client ID is provided, require all Google OAuth config fields
	if mainConfig.Auth.GoogleClientID != "" {
		missing := []string{}
		if mainConfig.Auth.GoogleClientSecret == "" {
			missing = append(missing, "auth.google_client_secret")
		}
		if mainConfig.Auth.GoogleRedirectURL == "" {
			missing = append(missing, "auth.google_redirect_url")
		}
		if mainConfig.Auth.GoogleSuccessRedirectURL == "" {
			missing = append(missing, "auth.google_success_redirect_url")
		}
		if mainConfig.Auth.GoogleFailureRedirectURL == "" {
			missing = append(missing, "auth.google_failure_redirect_url")
		}
		if len(missing) > 0 {
			logger.Fatal().Str("missing_fields", strings.Join(missing, ", ")).Msg("incomplete google oauth config")
		}
	}

	return mainConfig, nil
}

func validateFileStorageConfig(cfg *Config) error {
	if cfg == nil {
		return nil
	}

	switch cfg.FileStorage.Provider {
	case "":
		return fmt.Errorf("file storage provider is required")
	case "local", "s3":
	default:
		return fmt.Errorf("unsupported file storage provider: %s", cfg.FileStorage.Provider)
	}

	if cfg.FileStorage.MaxUploadSizeMB <= 0 {
		return fmt.Errorf("file storage max_upload_size_mb must be greater than 0")
	}

	if len(cfg.FileStorage.AllowedMimeTypes) == 0 {
		return fmt.Errorf("file storage allowed_mime_types cannot be empty")
	}

	if cfg.FileStorage.Provider == "local" {
		if strings.TrimSpace(cfg.FileStorage.Local.BaseDir) == "" {
			return fmt.Errorf("file storage local.base_dir is required")
		}
		if strings.TrimSpace(cfg.FileStorage.Local.PublicPath) == "" {
			return fmt.Errorf("file storage local.public_path is required")
		}
	}

	if cfg.FileStorage.Provider == "s3" {
		if strings.TrimSpace(cfg.FileStorage.S3.Bucket) == "" {
			return fmt.Errorf("file storage s3.bucket is required")
		}
		if strings.TrimSpace(cfg.FileStorage.S3.Region) == "" {
			return fmt.Errorf("file storage s3.region is required")
		}
	}

	return nil
}
