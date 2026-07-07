//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/matchlock/backend-go/internal/config"
	"github.com/matchlock/backend-go/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	gdb, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db: %v\n", err)
		os.Exit(1)
	}
	if err := db.Ping(context.Background(), gdb); err != nil {
		fmt.Fprintf(os.Stderr, "ping: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("database ok")
}