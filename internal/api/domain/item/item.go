package item

import "backend-service/internal/api/domain"

type Interactor struct {
	repository domain.ItemRepository
}

// NewInteractor создаёт доменный обработчик сценариев работы с записями
func NewInteractor(repository domain.ItemRepository) *Interactor {
	return &Interactor{repository: repository}
}
