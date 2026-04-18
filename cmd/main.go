package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/danilovaalina/calendar/internal/api"
	"github.com/danilovaalina/calendar/internal/config"
	"github.com/danilovaalina/calendar/internal/logger"
	"github.com/danilovaalina/calendar/internal/notifier"
	"github.com/danilovaalina/calendar/internal/repository"
	"github.com/danilovaalina/calendar/internal/service"
	"github.com/labstack/echo/v5"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	l := logger.New(logger.Options{
		FilePath:   cfg.LogFile,
		BufferSize: cfg.BufferSize,
	})
	go l.Run(ctx)

	svc := service.New(service.Options{
		Repo:       repository.New(),
		Client:     notifier.New(cfg.NotifierURL),
		BufferSize: cfg.BufferSize,
	})
	go svc.RunNotifier(ctx)
	go svc.RunCleanup(ctx, cfg.Interval)

	a := api.New(svc, l)

	sc := echo.StartConfig{
		Address:         cfg.Addr,
		GracefulTimeout: 10 * time.Second, // Даем время на закрытие соединений
	}

	if err = sc.Start(ctx, a); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Stack().Err(err).Send()
	}

	log.Info().Msg("server stopped")
}
