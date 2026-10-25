package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"

	"backend-service/configs"
	httpdelivery "backend-service/internal/api/delivery/http"
	"backend-service/internal/api/delivery/http/health"
	itemhttp "backend-service/internal/api/delivery/http/item"
	itemdomain "backend-service/internal/api/domain/item"
	"backend-service/internal/api/repositories/pg"
	httpserver "backend-service/pkg/api/http"
)

type App struct {
	Config *configs.Config
}

// NewApp создаёт приложение с загруженной конфигурацией
func NewApp(config *configs.Config) *App {
	return &App{Config: config}
}

// Start собирает зависимости, запускает HTTP-сервер и выполняет корректное завершение
func (a *App) Start(logger *slog.Logger) error {
	// Контекст отменяется при получении SIGINT или SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Создаём единственный пул соединений PostgreSQL на весь процесс
	repository := pg.NewRepository(a.Config.Infrastructure.Postgresql.RW)
	if err := repository.InitDB(ctx, logger); err != nil {
		logger.Error("database initialization failed", "error", err)
		return errors.New("database initialization failed")
	}
	defer func() {
		if err := repository.Close(); err != nil {
			logger.Error("database close failed", "error", err)
		}
	}()

	// Вручную связываем репозиторий, домен, обработчики и маршрутизатор
	itemInteractor := itemdomain.NewInteractor(repository)
	itemHandler := itemhttp.NewHandler(itemInteractor)
	healthHandler := health.NewHandler(
		repository.GetDB(),
		a.Config.Application.InstanceID,
		a.Config.Infrastructure.Postgresql.RW.ConnectTimeout.Duration,
	)
	router := httpdelivery.NewRouter(logger, a.Config, itemHandler, healthHandler)
	server := httpserver.NewServer(router.Handler(), a.Config.Listen.HTTP)

	// Запускаем сервер асинхронно, чтобы одновременно ожидать ошибку или сигнал остановки
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server starting", "instance_id", a.Config.Application.InstanceID, "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received", "instance_id", a.Config.Application.InstanceID)
	case serverErr := <-serverErrors:
		if !errors.Is(serverErr, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", serverErr)
		}
		return nil
	}

	// Ограничиваем время ожидания уже принятых HTTP-запросов
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.Config.Listen.HTTP.ShutdownTimeout.Duration)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	logger.Info("http server stopped", "instance_id", a.Config.Application.InstanceID)
	return nil
}
