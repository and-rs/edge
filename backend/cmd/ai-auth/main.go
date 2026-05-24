package main

import (
	"fmt"
	"os"

	"github.com/index/stint/backend/internal/app"
	"github.com/index/stint/backend/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ai-auth <login-openai|status-openai|logout-openai>")
		os.Exit(1)
	}

	cfg := config.Load()
	var err error

	switch os.Args[1] {
	case "login-openai":
		err = app.LoginOpenAI(cfg.AI, os.Stdout)
	case "status-openai":
		err = app.PrintOpenAIStatus(cfg.AI, os.Stdout)
	case "logout-openai":
		err = app.LogoutOpenAI(cfg.AI)
	default:
		err = fmt.Errorf("unknown command: %s", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
