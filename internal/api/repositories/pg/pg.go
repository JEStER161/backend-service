package pg

import (
	"context"
	"fmt"
	"log/slog"

	"backend-service/configs"
	"backend-service/pkg/migrate"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	config configs.PostgresqlInstance
	db     *sqlx.DB
}

// NewRepository создаёт PostgreSQL-репозиторий с параметрами подключения
func NewRepository(config configs.PostgresqlInstance) *Repository {
	return &Repository{config: config}
}

// InitDB открывает пул соединений и применяет включённые миграции
func (r *Repository) InitDB(ctx context.Context, logger *slog.Logger) error {
	return r.initDB(ctx, logger, true)
}

// InitDBWithoutMigrations открывает пул соединений без изменения схемы БД
func (r *Repository) InitDBWithoutMigrations(ctx context.Context) error {
	return r.initDB(ctx, nil, false)
}

// initDB настраивает пул, проверяет подключение и при необходимости обновляет схему
func (r *Repository) initDB(ctx context.Context, logger *slog.Logger, runMigrations bool) error {
	// sqlx.Open создаёт пул, но реальное подключение проверяется ниже через PingContext
	db, err := sqlx.Open("pgx", r.config.DSN())
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(r.config.Connections.MaxOpenConns)
	db.SetMaxIdleConns(r.config.Connections.MaxIdleConns)
	db.SetConnMaxLifetime(r.config.Connections.ConnMaxLifetime.Duration)
	db.SetConnMaxIdleTime(r.config.Connections.ConnMaxIdleTime.Duration)

	// Ограничиваем начальную проверку отдельным коротким таймаутом
	pingCtx, cancel := context.WithTimeout(ctx, r.config.ConnectTimeout.Duration)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return fmt.Errorf("ping database: %w", err)
	}
	r.db = db

	if runMigrations && r.config.Migrations.IsEnabled {
		if err := migrate.Up(logger, db.DB, r.config.Migrations.DirPath); err != nil {
			_ = db.Close()
			r.db = nil
			return err
		}
	}
	return nil
}

// GetDB возвращает общий пул соединений для health check
func (r *Repository) GetDB() *sqlx.DB {
	return r.db
}

// Close закрывает общий пул соединений PostgreSQL
func (r *Repository) Close() error {
	if r.db == nil {
		return nil
	}
	return r.db.Close()
}
