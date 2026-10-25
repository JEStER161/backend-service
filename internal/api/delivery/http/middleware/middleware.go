package middleware

import (
	stderrors "errors"
	"log/slog"
	"time"

	apierrors "backend-service/pkg/api/errors"

	"github.com/labstack/echo/v4"
)

const (
	InstanceIDHeader        = "X-Instance-ID"
	InternalErrorContextKey = "internal_error"
)

// InstanceID добавляет идентификатор реплики в каждый HTTP-ответ
func InstanceID(instanceID string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(InstanceIDHeader, instanceID)
			return next(c)
		}
	}
}

// RequestLogging записывает результат и длительность каждого HTTP-запроса
func RequestLogging(logger *slog.Logger, instanceID, requestIDHeader string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			started := time.Now()
			err := next(c)
			// Сначала передаём ошибку общему обработчику, чтобы в лог попал итоговый статус
			if err != nil {
				c.Error(err)
			}
			attributes := []any{
				"request_id", c.Response().Header().Get(requestIDHeader),
				"instance_id", instanceID,
				"method", c.Request().Method,
				"path", c.Path(),
				"status", c.Response().Status,
				"duration", time.Since(started),
			}
			if err != nil {
				var applicationError *apierrors.Error
				if stderrors.As(err, &applicationError) && applicationError.Internal != nil {
					attributes = append(attributes, "error", applicationError.Internal)
				} else {
					var echoError *echo.HTTPError
					if stderrors.As(err, &echoError) && echoError.Internal != nil {
						attributes = append(attributes, "error", echoError.Internal)
					} else {
						attributes = append(attributes, "error", err)
					}
				}
			} else if internalError, ok := c.Get(InternalErrorContextKey).(error); ok {
				attributes = append(attributes, "error", internalError)
			}
			logger.InfoContext(c.Request().Context(), "http request", attributes...)
			return nil
		}
	}
}
