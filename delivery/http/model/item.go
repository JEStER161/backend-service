package model

import "time"

type CreateItemRequest struct {
	Name  *string `json:"name"`
	Value *string `json:"value"`
}

type Item struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

type Health struct {
	Status     string `json:"status"`
	Database   string `json:"database"`
	InstanceID string `json:"instance_id"`
}

type ErrorResponse struct {
	Error Error `json:"error"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
