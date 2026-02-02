package main

import (
	"net/http"

	"github.com/danilovaalina/calendar/internal/api"
	"github.com/danilovaalina/calendar/internal/config"
	"github.com/danilovaalina/calendar/internal/repository"
	"github.com/danilovaalina/calendar/internal/service"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	a := api.New(service.New(repository.New()), api.WithLogFile(cfg.LogFile))

	err = http.ListenAndServe(cfg.Addr, a)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}
}
