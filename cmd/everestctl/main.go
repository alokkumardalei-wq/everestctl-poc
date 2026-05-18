package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/openeverest/everestctl-poc/internal/backend"
	"github.com/openeverest/everestctl-poc/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	be := backend.NewMemoryBackend()
	root := cli.NewRoot(be, os.Stdout, os.Stderr)
	if err := root.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
