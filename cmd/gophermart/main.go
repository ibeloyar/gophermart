package main

import (
	"log"

	"github.com/ibeloyar/gophermart/internal/app"
	"github.com/ibeloyar/gophermart/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(cfg); err != nil {
		log.Fatal(err)
	}
}
