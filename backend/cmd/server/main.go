package main

import (
	"log"

	"github.com/index/stint/backend/internal/app"
	"github.com/index/stint/backend/internal/config"
)

func main() {
	cfg := config.Load()
	server := app.NewServer(cfg)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
