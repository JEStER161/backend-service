package models

import (
	"time"

	"github.com/google/uuid"
)

type Item struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	Value     string    `db:"value"`
	CreatedAt time.Time `db:"created_at"`
}
