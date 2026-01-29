package config

import (
	"flag"
	"time"

	"github.com/caarlos0/env/v11"
)

const (
	DefaultRunAddress           = ":8080"
	DefaultDatabaseURI          = ""
	DefaultAccrualSystemAddress = "http://localhost:4000"
	DefaultPassCost             = 3
	DefaultSecretKey            = "secret"
	DefaultTokenLifetime        = 3 * time.Hour
)

type Config struct {
	RunAddress           string        `env:"RUN_ADDRESS"`
	DatabaseURI          string        `env:"DATABASE_URI"`
	AccrualSystemAddress string        `env:"ACCRUAL_SYSTEM_ADDRESS"`
	PassCost             int           `env:"PASS_COST"`
	SecretKey            string        `env:"SECRET_KEY"`
	TokenLifetime        time.Duration `env:"TOKEN_LIFETIME" default:"3h"`
}

func Read() (Config, error) {
	config := Config{}

	flag.StringVar(&config.RunAddress, "a", DefaultRunAddress, "Server run address")
	flag.StringVar(&config.DatabaseURI, "d", DefaultDatabaseURI, "Database connect string")
	flag.StringVar(&config.AccrualSystemAddress, "r", DefaultAccrualSystemAddress, "Accrual system address protocol://hostname:port")

	flag.IntVar(&config.PassCost, "p", DefaultPassCost, "Pass cost for password hash")
	flag.StringVar(&config.SecretKey, "s", DefaultSecretKey, "Secret key for token")
	flag.DurationVar(&config.TokenLifetime, "h", DefaultTokenLifetime, "Token lifetime (e.g. 1h, 30m, 2h30m)")

	flag.Parse()

	err := env.Parse(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}
