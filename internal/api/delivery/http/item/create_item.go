package item

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"backend-service/delivery/http/model"
	apierrors "backend-service/pkg/api/errors"

	"github.com/labstack/echo/v4"
)

// Create строго декодирует запрос, проверяет поля и создаёт запись
func (handler *Handler) Create(c echo.Context) error {
	var request model.CreateItemRequest
	decoder := json.NewDecoder(c.Request().Body)
	// Неизвестные поля считаются ошибкой, чтобы клиент не передавал опечатки незаметно
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		message := "request body must be a valid JSON object"
		if errors.Is(err, io.EOF) {
			message = "request body is required"
		}
		return apierrors.New(http.StatusBadRequest, "validation_error", message, err)
	}
	// После объекта допускаются только пробельные символы
	if err := ensureJSONEOF(decoder); err != nil {
		return apierrors.New(http.StatusBadRequest, "validation_error", "request body must contain exactly one JSON object", err)
	}
	name, value, err := validateCreateRequest(request)
	if err != nil {
		return err
	}
	created, err := handler.item.Create(c.Request().Context(), name, value)
	if err != nil {
		return apierrors.New(http.StatusInternalServerError, "internal_error", "internal server error", err)
	}
	return c.JSON(http.StatusCreated, toResponse(created))
}
