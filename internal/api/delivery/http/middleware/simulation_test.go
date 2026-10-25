package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apierrors "backend-service/pkg/api/errors"

	"github.com/labstack/echo/v4"
)

// TestSimulationDisabled проверяет отсутствие накладных действий при выключенной симуляции
func TestSimulationDisabled(t *testing.T) {
	called := false
	handler := SimulationWithRandom(false, time.Hour, 1, func() float64 { return 0 })(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusNoContent)
	})
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	if err := handler(c); err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if !called {
		t.Fatal("next handler was not called")
	}
}

// TestSimulationFailure проверяет генерацию искусственного отказа
func TestSimulationFailure(t *testing.T) {
	handler := SimulationWithRandom(true, 0, 0.5, func() float64 { return 0.25 })(func(echo.Context) error {
		t.Fatal("next handler must not be called")
		return nil
	})
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	err := handler(c)
	var applicationError *apierrors.Error
	if !errors.As(err, &applicationError) || applicationError.Code != "simulated_failure" || applicationError.Status != http.StatusServiceUnavailable {
		t.Fatalf("error = %#v", err)
	}
}

// TestSimulationShortDelay проверяет применение короткой задержки
func TestSimulationShortDelay(t *testing.T) {
	delay := 5 * time.Millisecond
	handler := SimulationWithRandom(true, delay, 0, func() float64 { return 1 })(func(echo.Context) error { return nil })
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	started := time.Now()
	if err := handler(c); err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if time.Since(started) < delay {
		t.Fatal("configured delay was not applied")
	}
}

// TestSimulationHonorsCancellation проверяет прерывание задержки при отмене контекста
func TestSimulationHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	e := echo.New()
	c := e.NewContext(request, httptest.NewRecorder())
	handler := SimulationWithRandom(true, time.Hour, 0, func() float64 { return 1 })(func(echo.Context) error {
		t.Fatal("next handler must not be called")
		return nil
	})
	started := time.Now()
	err := handler(c)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if time.Since(started) > 100*time.Millisecond {
		t.Fatal("cancellation was not handled promptly")
	}
}
