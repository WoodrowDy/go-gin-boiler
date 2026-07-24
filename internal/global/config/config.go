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
	// TODO(step-5):  DB  DB      // DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD
	// TODO(step-11): JWT JWT     // JWT_SECRET
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

	// TODO: appEnv 가 EnvLocal/Dev/Prod/Test 중 하나인지 검사 → 아니면 fmt.Errorf 반환
	switch appEnv {
	case EnvLocal, EnvDev, EnvProd, EnvTest:
	default:
		return nil, fmt.Errorf("invalid APP_ENV: %s", appEnv)
	}

	path := "envs/.env." + appEnv
	if err := godotenv.Load(path); err != nil {
		log.Printf("config: no env file at %s, using OS env", path)
	}

	// TODO: cfg 선언하고 env.Parse(&cfg) → 에러면 감싸서 반환
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env: %w", err)
	}

	// TODO: &cfg, nil 반환
	return cfg, nil

}
