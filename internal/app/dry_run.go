package app

import (
	"context"
	"errors"
	"log/slog"

	"backend-service/internal/api/repositories/pg"
)

// DryRun проверяет подключение к PostgreSQL без запуска HTTP-сервера
func (a *App) DryRun(logger *slog.Logger) error {
	repository := pg.NewRepository(a.Config.Infrastructure.Postgresql.RW)
	if err := repository.InitDBWithoutMigrations(context.Background()); err != nil {
		logger.Error("PostgreSQL dependency check failed", "error", err)
		return errors.New("PostgreSQL dependency check failed")
	}
	defer func() {
		if err := repository.Close(); err != nil {
			logger.Error("database close failed", "error", err)
		}
	}()
	logger.Info("dependency check completed", "postgresql", "ok")
	return nil
}
