package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Quran     QuranConfig
	JWT       JWTConfig
	Email     EmailConfig
	AppURL    string
	RateLimit RateLimitConfig
	Token     TokenConfig
}

type QuranConfig struct {
	Provider     string
	Environment  string
	ClientID     string
	ClientSecret string
	APIBaseURL   string
	OAuthBaseURL string
	Timeout      time.Duration
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	URL string
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	Issuer        string
}

type EmailConfig struct {
	Provider       string
	SMTPHost       string
	SMTPPort       int
	SMTPUser       string
	SMTPPass       string
	From           string
	FromName       string
	SendGridAPIKey string
	MailgunAPIKey  string
	MailgunDomain  string
}

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

type TokenConfig struct {
	AccessExpiration  time.Duration
	RefreshExpiration time.Duration
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/ilmnafi")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = viper.ReadInConfig()

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			URL:             viper.GetString("DATABASE_URL"),
			MaxOpenConns:    viper.GetInt("DATABASE_MAX_OPEN_CONNS"),
			MaxIdleConns:    viper.GetInt("DATABASE_MAX_IDLE_CONNS"),
			ConnMaxLifetime: getDuration("DATABASE_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			URL: viper.GetString("REDIS_URL"),
		},
		Quran: QuranConfig{
			Provider:     getEnv("QURAN_PROVIDER", "quran_foundation"),
			Environment:  getEnv("QURAN_FOUNDATION_ENV", "prelive"),
			ClientID:     viper.GetString("QURAN_FOUNDATION_CLIENT_ID"),
			ClientSecret: viper.GetString("QURAN_FOUNDATION_CLIENT_SECRET"),
			APIBaseURL:   viper.GetString("QURAN_FOUNDATION_API_BASE_URL"),
			OAuthBaseURL: viper.GetString("QURAN_FOUNDATION_OAUTH_BASE_URL"),
			Timeout:      getDuration("QURAN_PROVIDER_TIMEOUT", 15*time.Second),
		},
		JWT: JWTConfig{
			AccessSecret:  viper.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),
			Issuer:        viper.GetString("JWT_ISSUER"),
		},
		Email: EmailConfig{
			Provider:       viper.GetString("EMAIL_PROVIDER"),
			SMTPHost:       viper.GetString("EMAIL_SMTP_HOST"),
			SMTPPort:       viper.GetInt("EMAIL_SMTP_PORT"),
			SMTPUser:       viper.GetString("EMAIL_SMTP_USER"),
			SMTPPass:       viper.GetString("EMAIL_SMTP_PASS"),
			From:           viper.GetString("EMAIL_FROM"),
			FromName:       viper.GetString("EMAIL_FROM_NAME"),
			SendGridAPIKey: viper.GetString("SENDGRID_API_KEY"),
			MailgunAPIKey:  viper.GetString("MAILGUN_API_KEY"),
			MailgunDomain:  viper.GetString("MAILGUN_DOMAIN"),
		},
		AppURL: viper.GetString("APP_URL"),
		RateLimit: RateLimitConfig{
			Requests: viper.GetInt("RATE_LIMIT_REQUESTS"),
			Window:   getDuration("RATE_LIMIT_WINDOW", time.Minute),
		},
		Token: TokenConfig{
			AccessExpiration:  getDuration("ACCESS_TOKEN_EXPIRATION", 15*time.Minute),
			RefreshExpiration: getDuration("REFRESH_TOKEN_EXPIRATION", 7*24*time.Hour),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWT.AccessSecret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if c.JWT.RefreshSecret == "" {
		return fmt.Errorf("JWT_REFRESH_SECRET is required")
	}
	if c.Email.From == "" {
		return fmt.Errorf("EMAIL_FROM is required")
	}
	if c.AppURL == "" {
		return fmt.Errorf("APP_URL is required")
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := viper.GetString(key); val != "" {
		return val
	}
	return defaultVal
}

func getDuration(key string, defaultVal time.Duration) time.Duration {
	if val := viper.GetDuration(key); val > 0 {
		return val
	}
	return defaultVal
}
