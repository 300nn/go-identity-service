package httpserver

import (
	"net/http"
	"time"
)

type Config struct {
	Port string
}

func New(cfg Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
