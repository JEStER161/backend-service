package domain

import (
	"context"

	"backend-service/internal/api/models"

	"github.com/google/uuid"
)

// ItemRepository описывает операции хранения записей
type ItemRepository interface {
	// CreateItem сохраняет новую запись
	CreateItem(ctx context.Context, item models.Item) (models.Item, error)
	// GetItemByID получает запись по UUID
	GetItemByID(ctx context.Context, id uuid.UUID) (models.Item, error)
}

// Item описывает доступные бизнес-сценарии записей
type Item interface {
	// Create создаёт запись с переданными данными
	Create(ctx context.Context, name, value string) (models.Item, error)
	// GetByID получает запись по UUID
	GetByID(ctx context.Context, id uuid.UUID) (models.Item, error)
}
