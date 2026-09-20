package config

import (
	"fmt"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const RequestIDHeader = "x-request-id"

type CtxKeyRequestID struct{}

type Config struct {
	DomainURL  string           `koanf:"DOMAIN_URL" validate:"required"`
	HTTPServer HTTPServerConfig `koanf:",squash"`
	DB         DBConfig         `koanf:",squash"`
}

type HTTPServerConfig struct {
	Host    string        `koanf:"HTTP_SERVER_HOST" validate:"required"`
	Port    uint16        `koanf:"HTTP_SERVER_PORT" validate:"required"`
	Timeout time.Duration `koanf:"HTTP_SERVER_TIMEOUT" validate:"required"`
}

type DBConfig struct {
	Host           string        `koanf:"DB_HOST" validate:"required"`
	Port           int           `koanf:"DB_PORT" validate:"required"`
	User           string        `koanf:"DB_USER" validate:"required"`
	Password       string        `koanf:"DB_PASSWORD" validate:"required"`
	Name           string        `koanf:"DB_NAME" validate:"required"`
	SSLMode        string        `koanf:"DB_SSL_MODE" validate:"required"`
	RequestTimeout time.Duration `koanf:"DB_REQUEST_TIMEOUT" validate:"required"`
	URLsTTL        time.Duration `koanf:"DB_URLS_TTL" validate:"required"`
	ClearFrequency time.Duration `koanf:"DB_CLEAR_FREQUENCY" validate:"required"`
}

func (c DBConfig) ConnString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

func LoadConfig() *Config {
	k := koanf.New(".")

	for _, path := range []string{".env", "config/.env"} {
		if err := k.Load(file.Provider(path), dotenv.Parser()); err != nil && !os.IsNotExist(err) {
			panic(fmt.Sprintf("failed to read %s: %v", path, err))
		}
	}

	if err := k.Load(env.Provider(".", env.Opt{}), nil); err != nil {
		panic(fmt.Sprintf("failed to load env variables: %v", err))
	}

	var config Config

	if err := k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{
		Tag:       "koanf",
		FlatPaths: true,
	}); err != nil {
		panic(fmt.Sprintf("failed to parse configuration: %v", err))
	}

	validate := validator.New()
	if err := validate.Struct(&config); err != nil {
		panic(fmt.Sprintf("invalid configuration: %s", err))
	}

	return &config
}
