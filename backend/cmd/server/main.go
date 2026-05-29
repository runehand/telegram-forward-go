package main

import (
	"fmt"
	"os"

	"zenfl-forwarder/backend/internal/app"
	"zenfl-forwarder/backend/internal/config"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	if err := app.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "app error: %v\n", err)
		os.Exit(1)
	}
}
