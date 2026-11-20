package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	App    App
	Server Server
	DB     DB // db.go
	// TODO(step-12): JWT JWT     // JWT_SECRET
}

const (
	EnvLocal = "local"
	EnvDev   = "dev"
	EnvProd  = "prod"
	EnvTest  = "test"
)

type App struct {
	Env string `env:"APP_ENV" envDefault:"local"`
}

type Server struct {
	Port              int           `env:"PORT,required"`
	ReadHeaderTimeout time.Duration `env:"SERVER_READ_HEADER_TIMEOUT" envDefault:"5s"`
	ReadTimeout       time.Duration `env:"SERVER_READ_TIMEOUT"        envDefault:"5s"`
	WriteTimeout      time.Duration `env:"SERVER_WRITE_TIMEOUT"       envDefault:"10s"`
	IdleTimeout       time.Duration `env:"SERVER_IDLE_TIMEOUT"        envDefault:"15s"`
	ShutdownTimeout   time.Duration `env:"SERVER_SHUTDOWN_TIMEOUT"    envDefault:"10s"`
}

func (s Server) Addr() string {
	return ":" + strconv.Itoa(s.Port)
}

func Load() (*Config, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = EnvLocal
	}

	switch appEnv {
	case EnvLocal, EnvDev, EnvProd, EnvTest:
	default:
		return nil, fmt.Errorf("invalid APP_ENV: %s", appEnv)
	}

	path := "envs/.env." + appEnv
	if err := godotenv.Load(path); err != nil {
		log.Printf("config: no env file at %s, using OS env", path)
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env: %w", err)
	}

	return cfg, nil
}
