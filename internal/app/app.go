package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/savioruz/goth/config"
)

//go:generate go run github.com/google/wire/cmd/wire

func Run(cfg *config.Config) {
	app, err := InitializeApp(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize application: %v", err))
	}

	defer app.PG.Pool.Close()
	defer app.Redis.Close()

	if err := app.PG.Ping(context.Background()); err != nil {
		app.Logger.Fatal(fmt.Errorf("app - Run - postgres.Ping: %w", err))
	}

	if err := app.Redis.Ping(context.Background()); err != nil {
		app.Logger.Fatal(fmt.Errorf("app - Run - redis.Ping: %w", err))
	}

	app.HTTPServer.Start()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		app.Logger.Info("app - Run - signal: " + s.String())
	case err = <-app.HTTPServer.Notify():
		app.Logger.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
	}

	err = app.HTTPServer.Shutdown()
	if err != nil {
		app.Logger.Error(fmt.Errorf("app - Run - httpServer.Shutdown: %w", err))
	}
}
