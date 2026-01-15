package config

//- адрес и порт запуска сервиса: переменная окружения ОС `RUN_ADDRESS` или флаг `-a`
//- адрес подключения к базе данных: переменная окружения ОС `DATABASE_URI` или флаг `-d`
//- адрес системы расчёта начислений: переменная окружения ОС `ACCRUAL_SYSTEM_ADDRESS` или флаг `-r`

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

const (
	DefaultRunAddress           = ":8080"
	DefaultDatabaseURI          = ""
	DefaultAccrualSystemAddress = "./accrual/accrual_linux_amd64"
)

type Config struct {
	RunAddress           string `env:"RUN_ADDRESS"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

func Read() (Config, error) {
	config := Config{}

	flag.StringVar(&config.RunAddress, "a", DefaultRunAddress, "")
	flag.StringVar(&config.DatabaseURI, "d", DefaultDatabaseURI, "Database connect string")
	flag.StringVar(&config.AccrualSystemAddress, "r", DefaultAccrualSystemAddress, "")

	flag.Parse()

	err := env.Parse(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}
