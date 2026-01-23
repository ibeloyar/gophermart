package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

const (
	DefaultRunAddress           = ":8080"
	DefaultDatabaseURI          = ""
	DefaultAccrualSystemAddress = "http://localhost:4000"
	DefaultPassCost             = 3
	DefaultSecretKey            = "secret"
	DefaultTokenLifetimeHours   = 3
)

type Config struct {
	RunAddress           string `env:"RUN_ADDRESS"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	PassCost             int    `env:"PASS_COST"`
	SecretKey            string `env:"SECRET_KEY"`
	TokenLifetimeHours   int    `env:"TOKEN_LIFETIME_HOURS"`
}

func Read() (Config, error) {
	config := Config{}

	flag.StringVar(&config.RunAddress, "a", DefaultRunAddress, "Server run address")
	flag.StringVar(&config.DatabaseURI, "d", DefaultDatabaseURI, "Database connect string")
	flag.StringVar(&config.AccrualSystemAddress, "r", DefaultAccrualSystemAddress, "Accrual system address protocol://hostname:port")

	flag.IntVar(&config.PassCost, "p", DefaultPassCost, "Pass cost for password hash")
	flag.StringVar(&config.SecretKey, "s", DefaultSecretKey, "Secret key for token")
	flag.IntVar(&config.TokenLifetimeHours, "h", DefaultTokenLifetimeHours, "Token lifetime in hours")

	flag.Parse()

	err := env.Parse(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}
