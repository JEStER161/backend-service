package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"backend-service/internal/api/domain"
	"backend-service/internal/api/models"

	"github.com/google/uuid"
)

const (
	createItemQuery = `
		INSERT INTO items (id, name, value)
		VALUES ($1, $2, $3)
		RETURNING id, name, value, created_at`
	getItemByIDQuery = `
		SELECT id, name, value, created_at
		FROM items
		WHERE id = $1`
)

// CreateItem сохраняет запись и возвращает данные вместе со временем создания
func (r *Repository) CreateItem(ctx context.Context, value models.Item) (models.Item, error) {
	var created models.Item
	err := r.db.QueryRowxContext(ctx, createItemQuery, value.ID, value.Name, value.Value).StructScan(&created)
	if err != nil {
		return models.Item{}, fmt.Errorf("create item: %w", err)
	}
	return created, nil
}

// GetItemByID получает запись по UUID и преобразует отсутствие строки в доменную ошибку
func (r *Repository) GetItemByID(ctx context.Context, id uuid.UUID) (models.Item, error) {
	var found models.Item
	if err := r.db.GetContext(ctx, &found, getItemByIDQuery, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Item{}, domain.ErrItemNotFound
		}
		return models.Item{}, fmt.Errorf("get item by id: %w", err)
	}
	return found, nil
}
