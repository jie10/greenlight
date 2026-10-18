package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	App       AppConfig       `envPrefix:"APP_"`
	DB        DBConfig        `envPrefix:"DB_"`
	JWT       JWTConfig       `envPrefix:"JWT_"`
	SMTP      SMTPConfig      `envPrefix:"SMTP_"`
	OAuth2    OAuth2Config    `envPrefix:"OAUTH2_"`
	CORS      CORSConfig      `envPrefix:"CORS_"`
	RateLimit RateLimitConfig `envPrefix:"RATE_LIMIT_"`
}

type AppConfig struct {
	Env          string        `env:"ENV" envDefault:"development"`
	Port         string        `env:"PORT" envDefault:"3001"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout  time.Duration `env:"IDLE_TIMEOUT" envDefault:"120s"`
}

type DBConfig struct {
	Host     string `env:"HOST"`
	Port     string `env:"PORT"`
	User     string `env:"USER"`
	Password string `env:"PASSWORD"`
	Name     string `env:"NAME"`
	SSLMode  string `env:"SSLMODE" envDefault:"disable"`
	MaxConns int32  `env:"MAX_CONNS" envDefault:"100"`
	MinConns int32  `env:"MIN_CONNS" envDefault:"10"`
	DNS      string `env:"-"`
}

type JWTConfig struct {
	Secret        string        `env:"SECRET"`
	AccessExpiry  time.Duration `env:"ACCESS_EXPIRY" envDefault:"15m"`
	RefreshExpiry time.Duration `env:"REFRESH_EXPIRY" envDefault:"168h"`
}

type SMTPConfig struct {
	Host     string `env:"HOST" envDefault:"localhost"`
	Port     int    `env:"PORT" envDefault:"587"`
	Username string `env:"USERNAME"`
	Password string `env:"PASSWORD"`
	From     string `env:"FROM"`
}

type OAuth2Config struct {
	ClientID     string `env:"CLIENT_ID"`
	ClientSecret string `env:"CLIENT_SECRET"`
	AuthURL      string `env:"AUTH_URL"`
	TokenURL     string `env:"Token_URL"`
}

type CORSConfig struct {
	AllowedOrigins []string `env:"ALLOWED_ORIGINS" envSeparator:","`
}

type RateLimitConfig struct {
	RPS   float64 `env:"RPS" envDefault:"100"`
	Burst int     `env:"BURST" envDefault:"200"`
}

func Load() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: parse environment: %w", err)
	}

	cfg.normalize()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	dsn, err := cfg.DB.BuildDSN()
	if err != nil {
		return nil, err
	}
	cfg.DB.DNS = dsn

	return cfg, nil
}

func (c *Config) normalize() {
	c.CORS.AllowedOrigins = trimAll(c.CORS.AllowedOrigins)
}

func (c *Config) validate() error {
	var errs []error

	for _, f := range []struct{ name, val string }{
		{"DB_HOST", c.DB.Host},
		{"DB_PORT", c.DB.Port},
		{"DB_USER", c.DB.User},
		{"DB_NAME", c.DB.Name},
	} {
		if f.val == "" {
			errs = append(errs, fmt.Errorf("config: required variable %s is empty", f.name))
		}
	}

	if c.JWT.Secret == "" {
		errs = append(errs, errors.New("config: JWT_SECRET must not be empty"))
	}

	if c.App.Env == "production" && c.JWT.Secret == "change-me" {
		errs = append(errs, errors.New("config: JWT_SECRET must not be the default in production"))
	}

	if c.RateLimit.RPS <= 0 || c.RateLimit.Burst <= 0 {
		errs = append(errs, errors.New("config: rate limit RPS and BURST must be positive"))
	}

	return errors.Join(errs...)
}

func (c DBConfig) BuildDSN() (string, error) {
	if c.Port == "" {
		return "", errors.New("config: DB_PORT must not be empty")
	}

	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   net.JoinHostPort(c.Host, c.Port),
		Path:   "/" + c.Name,
	}

	q := url.Values{}
	q.Set("sslmode", c.SSLMode)
	if c.MaxConns > 0 {
		q.Set("pool_max_conns", strconv.Itoa(int(c.MaxConns)))
	}
	if c.MinConns > 0 {
		q.Set("pool_min_conns", strconv.Itoa(int(c.MinConns)))
	}

	u.RawQuery = q.Encode()

	return u.String(), nil
}

func trimAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
