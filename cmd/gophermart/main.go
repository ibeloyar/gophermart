package main

import (
	"log"

	"github.com/ibeloyar/gophermart/internal/app"
	"github.com/ibeloyar/gophermart/internal/config"
	"github.com/ibeloyar/gophermart/pgk/logger"
)

func main() {
	lg, err := logger.New()
	if err != nil {
		log.Fatal(err)
	}
	defer lg.Sync()

	cfg, err := config.Read()
	if err != nil {
		lg.Fatal(err)
	}

	if err := app.Run(cfg, lg); err != nil {
		log.Fatal(err)
	}
}
