package main

import (
    "log"

    "github.com/index/stint/backend/internal/app"
    "github.com/index/stint/backend/internal/config"
)

func main() {
    var cfg config.Config = config.Load()
    var server *app.Server = app.NewServer(cfg)

    if err := server.Run(); err != nil {
        log.Fatal(err)
    }
}
