package config

import (
	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/user"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

var configValidator = validator.New(
	validator.WithRequiredStructEnabled())

type Config struct {
	HTTP      HTTPConfig           `yaml:"http" env-prefix:"HTTP_"`
	Database  DatabaseConfig       `yaml:"database" env-prefix:"DATABASE_"`
	Log       LogConfig            `yaml:"log" env-prefix:"LOG_"`
	App       AppConfig            `yaml:"app" env-prefix:"APP_"`
	Auth      AuthConfig           `yaml:"auth" env-prefix:"AUTH_"`
	RateLimit auth.RateLimitConfig `yaml:"rate_limit" env-prefix:"RATE_LIMIT_"`
	Redis     RedisConfig          `yaml:"redis" env-prefix:"REDIS_"`
	Cache     user.CacheConfig     `yaml:"cache" env-prefix:"CACHE_"`
}

type HTTPConfig struct {
	Host              string        `yaml:"host" env:"HOST" env-default:"0.0.0.0" validate:"required"`
	Port              int           `yaml:"port" env:"PORT" env-default:"8080" validate:"gte=1,lte=65535"`
	ReadTimeout       time.Duration `yaml:"read_timeout" env:"READ_TIMEOUT" env-default:"5s" validate:"gt=0"`
	WriteTimeout      time.Duration `yaml:"write_timeout" env:"WRITE_TIMEOUT" env-default:"10s" validate:"gt=0"`
	IdleTimeout       time.Duration `yaml:"idle_timeout" env:"IDLE_TIMEOUT" env-default:"1m" validate:"gt=0"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout" env:"READ_HEADER_TIMEOUT" env-default:"5s" validate:"gt=0"`
}

type DatabaseConfig struct {
	Host     string     `yaml:"host" env:"HOST" env-default:"localhost" validate:"required"`
	Port     int        `yaml:"port" env:"PORT" env-default:"5432" validate:"gte=1,lte=65535"`
	Name     string     `yaml:"name" env:"NAME" env-default:"go_crud" validate:"required"`
	User     string     `yaml:"user" env:"USER" env-default:"go_crud" validate:"required"`
	Password string     `yaml:"password" env:"PASSWORD"`
	Pool     PoolConfig `yaml:"pool" env-prefix:"POOL_"`
	SSL      string     `yaml:"ssl" env:"SSL" env-default:"disable"`
}

type PoolConfig struct {
	MaxCons           int32         `yaml:"max_cons" env:"MAX_CONS" env-default:"10" validate:"gt=0"`
	MinCons           int32         `yaml:"min_cons" env:"MIN_CONS" env-default:"10" validate:"gt=0"`
	MaxConLifeTime    time.Duration `yaml:"max_con_life_time" env:"MAX_CON_LIFE_TIME" env-default:"1h" validate:"gt=0"`
	MaxConIdleTime    time.Duration `yaml:"max_con_idle_time" env:"MAX_CON_IDLE_TIME" env-default:"20m" validate:"gt=0"`
	HealthCheckPeriod time.Duration `yaml:"health_check_period" env:"HEALTH_CHECK_PERIOD" env-default:"1m" validate:"gt=0"`
}

type LogConfig struct {
	Level string `yaml:"level" env:"LEVEL" env-default:"info" validate:"oneof=debug info warn error"`
}

type AppConfig struct {
	Name        string `yaml:"name" env:"NAME" env-default:"go-crud-api" validate:"required"`
	Version     string `yaml:"version" env:"VERSION" env-default:"dev" validate:"required"`
	Environment string `yaml:"environment" env:"ENVIRONMENT" env-default:"local" validate:"oneof=local development staging production"`
}

type AuthConfig struct {
	JWTSecret       string        `yaml:"jwt_secret" env:"JWT_SECRET" env-default:"local-dev-secret-change-me-please-32" validate:"required,min=32"`
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl" env:"ACCESS_TOKEN_TTL" env-default:"15m" validate:"gt=0"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl" env:"REFRESH_TOKEN_TTL" env-default:"720h" validate:"gt=0"`
}

type RedisConfig struct {
	Host     string `yaml:"host" env:"HOST" env-default:"localhost" validate:"required"`
	Port     int    `yaml:"port" env:"PORT" env-default:"6379" validate:"gte=1,lte=65535"`
	Password string `yaml:"password" env:"PASSWORD"`
	DB       int    `yaml:"db" env:"DB" env-default:"0" validate:"gte=0"`
}

func (r RedisConfig) Address() string {
	return net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
}

func (d *DatabaseConfig) DatabaseUrl() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   net.JoinHostPort(d.Host, strconv.Itoa(d.Port)),
		Path:   d.Name,
	}
	return u.String()
}

func (d *DatabaseConfig) DatabaseUrlWithSSL() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   net.JoinHostPort(d.Host, strconv.Itoa(d.Port)),
		Path:   d.Name,
	}
	q := u.Query()
	q.Set("sslmode", d.SSL)
	u.RawQuery = q.Encode()

	return u.String()
}

func (h *HTTPConfig) Address() string {
	return net.JoinHostPort(h.Host, strconv.Itoa(h.Port))
}

func Load(filename string) (*Config, error) {
	var cfg Config

	// Считываем config.yml
	if err := readConfig(filename, &cfg); err != nil {
		return nil, err
	}

	// Валидация, прописанная в Config (validate)
	if err := configValidator.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("validate config fields: %w", err)
	}

	// Дополнительная сложная валидация, при необходимости
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config rules: %w", err)
	}

	return &cfg, nil
}

// Считываем config.yml и переводим его в Config
func readConfig(filename string, cfg *Config) error {
	// Проверяем есть ли файл
	_, err := os.Stat(filename)

	switch {
	// Если файл есть, то пытаемся его считать, при ошибке, пробрасываем ее
	case err == nil:
		if err := cleanenv.ReadConfig(filename, cfg); err != nil {
			return fmt.Errorf("read config %q: %w", filename, err)
		}
		// Выставляем дефолтные значения, если файла нет
	case errors.Is(err, os.ErrNotExist):
		if err := cleanenv.ReadEnv(cfg); err != nil {
			return fmt.Errorf("read environment config: %w", err)
		}

		// Пробрасываем неизвестные ошибки
	default:
		return fmt.Errorf("check config %q: %w", filename, err)
	}

	return nil
}

func LoadDatabase(filename string) (*DatabaseConfig, error) {
	var cfg struct {
		Database DatabaseConfig `yaml:"database" env-prefix:"DATABASE_"`
	}

	_, err := os.Stat(filename)

	switch {
	// Если файл есть, то пытаемся его считать, при ошибке, пробрасываем ее
	case err == nil:
		if err := cleanenv.ReadConfig(filename, &cfg); err != nil {
			return nil, fmt.Errorf("read config %q: %w", filename, err)
		}
		// Выставляем дефолтные значения, если файла нет
	case errors.Is(err, os.ErrNotExist):
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("read environment config: %w", err)
		}

		// Пробрасываем неизвестные ошибки
	default:
		return nil, fmt.Errorf("check config %q: %w", filename, err)
	}

	if err := configValidator.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("validate config fields: %w", err)
	}

	return &cfg.Database, nil
}

func (c Config) Validate() error {
	if c.App.Environment == "production" &&
		c.Database.Password == "" {
		return errors.New(
			"database password is required in production",
		)
	}

	if c.App.Environment == "production" &&
		c.Auth.JWTSecret == "local-dev-secret-change-me-please-32" {
		return errors.New("auth jwt secret must be changed in production")
	}

	return nil
}
