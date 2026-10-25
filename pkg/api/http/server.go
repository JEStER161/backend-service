package http

import (
	"net/http"

	"backend-service/configs"
)

// NewServer создаёт HTTP-сервер с таймаутами из конфигурации
func NewServer(handler http.Handler, config configs.HTTP) *http.Server {
	return &http.Server{
		Addr:              config.Address(),
		Handler:           handler,
		ReadHeaderTimeout: config.ReadHeaderTimeout.Duration,
		ReadTimeout:       config.ReadTimeout.Duration,
		WriteTimeout:      config.WriteTimeout.Duration,
		IdleTimeout:       config.IdleTimeout.Duration,
	}
}
