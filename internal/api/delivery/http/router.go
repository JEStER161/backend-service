package http

import (
	stderrors "errors"
	"log/slog"
	"net/http"

	"backend-service/configs"
	"backend-service/delivery/http/model"
	"backend-service/internal/api/delivery/http/health"
	"backend-service/internal/api/delivery/http/item"
	appmiddleware "backend-service/internal/api/delivery/http/middleware"
	apierrors "backend-service/pkg/api/errors"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

type Router struct {
	e *echo.Echo
}

// NewRouter настраивает Echo, общие middleware и маршруты сервиса
func NewRouter(logger *slog.Logger, config *configs.Config, itemHandler *item.Handler, healthHandler *health.Handler) *Router {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = errorHandler

	// Общие middleware применяются ко всем ответам включая ошибки и неизвестные маршруты
	e.Use(echomiddleware.RecoverWithConfig(echomiddleware.RecoverConfig{
		DisablePrintStack:   true,
		DisableErrorHandler: true,
	}))
	e.Use(echomiddleware.RequestIDWithConfig(echomiddleware.RequestIDConfig{
		TargetHeader: config.Logging.RequestIDFieldName,
	}))
	e.Use(appmiddleware.InstanceID(config.Application.InstanceID))
	e.Use(appmiddleware.RequestLogging(logger, config.Application.InstanceID, config.Logging.RequestIDFieldName))
	e.Use(echomiddleware.ContextTimeout(config.Listen.HTTP.MaxRequestProcessingTime.Duration))

	// Health check регистрируется вне группы с имитацией деградации
	e.GET("/health", healthHandler.Check)
	api := e.Group("/api/v1")
	// Задержка и искусственные ошибки действуют только на прикладной API
	api.Use(appmiddleware.Simulation(
		config.Application.Simulation.IsEnabled,
		config.Application.Simulation.Delay.Duration,
		config.Application.Simulation.ErrorRate,
	))
	api.POST("/items", itemHandler.Create)
	api.GET("/items/:id", itemHandler.GetByID)
	return &Router{e: e}
}

// Handler возвращает Echo как стандартный HTTP-обработчик
func (r *Router) Handler() http.Handler {
	return r.e
}

// errorHandler преобразует внутренние ошибки в единый безопасный JSON-формат
func errorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "internal server error"

	// Прикладные ошибки уже содержат ожидаемый статус, код и публичное сообщение
	var applicationError *apierrors.Error
	if stderrors.As(err, &applicationError) {
		status, code, message = applicationError.Status, applicationError.Code, applicationError.Message
	} else {
		var echoError *echo.HTTPError
		if stderrors.As(err, &echoError) {
			status = echoError.Code
			switch status {
			case http.StatusNotFound:
				code, message = "not_found", "resource not found"
			case http.StatusMethodNotAllowed:
				code, message = "method_not_allowed", "method not allowed"
			case http.StatusBadRequest:
				code, message = "bad_request", "bad request"
			case http.StatusServiceUnavailable:
				code, message = "service_unavailable", "service temporarily unavailable"
			}
		}
	}
	if writeErr := c.JSON(status, model.ErrorResponse{Error: model.Error{Code: code, Message: message}}); writeErr != nil {
		c.Echo().Logger.Error(writeErr)
	}
}
