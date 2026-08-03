package httpserver

import (
	"CrudTutorialProject/internal/config"
	"net/http"
)

func New(cfg config.HTTPConfig, handler http.Handler) *http.Server {

	return &http.Server{
		Addr:         cfg.Address(),
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}
