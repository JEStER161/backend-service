package item

import (
	"context"

	"backend-service/internal/api/models"

	"github.com/google/uuid"
)

// Create создаёт UUID и передаёт новую запись в репозиторий
func (i *Interactor) Create(ctx context.Context, name, value string) (models.Item, error) {
	return i.repository.CreateItem(ctx, models.Item{
		ID:    uuid.New(),
		Name:  name,
		Value: value,
	})
}
