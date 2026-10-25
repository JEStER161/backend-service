package migrate

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Up применяет все ещё не выполненные SQL-миграции к переданной базе данных
func Up(logger *slog.Logger, db *sql.DB, migrationPath string) error {
	// Создаём драйвер поверх уже открытого сервисом пула соединений
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create migration database driver: %w", err)
	}

	// Файловый источник читает парные up и down миграции из указанного каталога
	runner, err := migrate.NewWithDatabaseInstance("file://"+migrationPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	// PostgreSQL advisory lock защищает от одновременного применения миграций репликами
	if err := runner.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("database migrations are already up to date")
			return nil
		}
		return fmt.Errorf("apply database migrations: %w", err)
	}

	version, dirty, err := runner.Version()
	if err != nil {
		return fmt.Errorf("read database migration version: %w", err)
	}
	logger.Info("database migrations applied", "version", version, "dirty", dirty)
	return nil
}
