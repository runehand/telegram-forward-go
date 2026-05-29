package app

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"zenfl-forwarder/internal/config"
	"zenfl-forwarder/internal/telegram/forwarder"
)

func Run(cfg config.Config) error {
	logger, err := zap.NewProduction()
	if err != nil {
		return err
	}
	defer func() { _ = logger.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	svc := forwarder.NewService(cfg.Telegram, logger)
	if err := svc.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}