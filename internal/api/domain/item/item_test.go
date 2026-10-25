package item

import (
	"context"
	"errors"
	"testing"

	"backend-service/internal/api/models"

	"github.com/google/uuid"
)

type fakeRepository struct {
	create func(context.Context, models.Item) (models.Item, error)
	get    func(context.Context, uuid.UUID) (models.Item, error)
}

// CreateItem передаёт вызов тестовой функции создания
func (f fakeRepository) CreateItem(ctx context.Context, value models.Item) (models.Item, error) {
	return f.create(ctx, value)
}

// GetItemByID передаёт вызов тестовой функции чтения
func (f fakeRepository) GetItemByID(ctx context.Context, id uuid.UUID) (models.Item, error) {
	return f.get(ctx, id)
}

// TestCreateGeneratesID проверяет генерацию UUID в доменном слое
func TestCreateGeneratesID(t *testing.T) {
	interactor := NewInteractor(fakeRepository{
		create: func(_ context.Context, value models.Item) (models.Item, error) {
			if value.ID == uuid.Nil {
				t.Fatal("expected generated UUID")
			}
			return value, nil
		},
	})
	created, err := interactor.Create(context.Background(), "name", "value")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Name != "name" || created.Value != "value" {
		t.Fatalf("Create() = %#v", created)
	}
}

// TestGetByIDPropagatesError проверяет передачу ошибки репозитория
func TestGetByIDPropagatesError(t *testing.T) {
	want := errors.New("repository failure")
	id := uuid.New()
	interactor := NewInteractor(fakeRepository{
		get: func(_ context.Context, got uuid.UUID) (models.Item, error) {
			if got != id {
				t.Fatalf("GetByID() id = %v, want %v", got, id)
			}
			return models.Item{}, want
		},
	})
	_, err := interactor.GetByID(context.Background(), id)
	if !errors.Is(err, want) {
		t.Fatalf("GetByID() error = %v, want %v", err, want)
	}
}
