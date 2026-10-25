package item

import (
	"context"

	"backend-service/internal/api/models"

	"github.com/google/uuid"
)

// GetByID получает запись из репозитория по UUID
func (i *Interactor) GetByID(ctx context.Context, id uuid.UUID) (models.Item, error) {
	return i.repository.GetItemByID(ctx, id)
}
