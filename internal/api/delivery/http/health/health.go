package health

import (
	"context"
	"net/http"
	"time"

	"backend-service/delivery/http/model"
	appmiddleware "backend-service/internal/api/delivery/http/middleware"

	"github.com/labstack/echo/v4"
)

// Checker описывает проверку доступности внешней зависимости
type Checker interface {
	// PingContext проверяет соединение с учётом контекста
	PingContext(ctx context.Context) error
}

// Handler обслуживает проверку состояния приложения и базы данных
type Handler struct {
	checker    Checker
	instanceID string
	timeout    time.Duration
}

// NewHandler создаёт обработчик health check с отдельным таймаутом БД
func NewHandler(checker Checker, instanceID string, timeout time.Duration) *Handler {
	return &Handler{checker: checker, instanceID: instanceID, timeout: timeout}
}

// Check проверяет PostgreSQL и возвращает состояние текущей реплики
func (h *Handler) Check(c echo.Context) error {
	// Проверка БД не должна зависать дольше настроенного connect timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()
	if err := h.checker.PingContext(ctx); err != nil {
		c.Set(appmiddleware.InternalErrorContextKey, err)
		return c.JSON(http.StatusServiceUnavailable, model.Health{
			Status:     "unavailable",
			Database:   "unavailable",
			InstanceID: h.instanceID,
		})
	}
	return c.JSON(http.StatusOK, model.Health{
		Status:     "ok",
		Database:   "ok",
		InstanceID: h.instanceID,
	})
}
