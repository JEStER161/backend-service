package item

import (
	"errors"
	"net/http"

	"backend-service/internal/api/domain"
	apierrors "backend-service/pkg/api/errors"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// GetByID проверяет UUID и возвращает запись или подходящую HTTP-ошибку
func (handler *Handler) GetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return apierrors.New(http.StatusBadRequest, "invalid_id", "id must be a valid UUID", err)
	}
	found, err := handler.item.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrItemNotFound) {
			return apierrors.New(http.StatusNotFound, "not_found", "item not found", err)
		}
		return apierrors.New(http.StatusInternalServerError, "internal_error", "internal server error", err)
	}
	return c.JSON(http.StatusOK, toResponse(found))
}
