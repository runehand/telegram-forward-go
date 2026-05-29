package app

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"zenfl-forwarder/backend/internal/config"
	"zenfl-forwarder/backend/internal/platform/logx"
	"zenfl-forwarder/backend/internal/restapi"
	"zenfl-forwarder/backend/internal/store/mongo"
	"zenfl-forwarder/backend/internal/telegram/forwarder"
)

func Run(cfg config.Config) error {
	logger, err := logx.New()
	if err != nil { return err }
	defer func() { _ = logger.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := mongo.New(ctx, cfg.Mongo)
	if err != nil { return err }
	defer func() { _ = store.Close(context.Background()) }()

	api := restapi.New(cfg, logger, store)
	tg := forwarder.NewService(cfg.Telegram, logger, store)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	wg.Add(2)
	go runUnit(ctx, &wg, errCh, api.Run)
	go runUnit(ctx, &wg, errCh, tg.Run)

	select {
	case <-ctx.Done():
		wg.Wait()
		return nil
	case e := <-errCh:
		stop()
		wg.Wait()
		if errors.Is(e, context.Canceled) { return nil }
		return e
	}
}

func runUnit(ctx context.Context, wg *sync.WaitGroup, errCh chan<- error, fn func(context.Context) error) {
	defer wg.Done()
	if err := fn(ctx); err != nil && !errors.Is(err, context.Canceled) {
		errCh <- err
	}
}
