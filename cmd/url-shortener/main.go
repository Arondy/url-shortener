package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Arondy/url-shortener/internal/args"
	"github.com/Arondy/url-shortener/internal/config"
	"github.com/Arondy/url-shortener/internal/core"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := config.GetLogger()
	defer config.CloseLoggerFiles()
	defer logger.Sync()

	cfg := config.LoadConfig()
	args := args.Parse()

	if err := core.Run(ctx, cfg, args, logger); err != nil {
		logger.Named("Main").Errorf("server error: %s", err)
		os.Exit(1)
	}
}
