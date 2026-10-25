package middleware

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"math"
	"net/http"
	"time"

	apierrors "backend-service/pkg/api/errors"

	"github.com/labstack/echo/v4"
)

// RandomFloat возвращает случайное число от нуля до единицы
type RandomFloat func() float64

// Simulation создаёт middleware управляемой задержки и искусственных ошибок
func Simulation(enabled bool, delay time.Duration, errorRate float64) echo.MiddlewareFunc {
	return SimulationWithRandom(enabled, delay, errorRate, secureRandomFloat)
}

// SimulationWithRandom позволяет подменить генератор случайных чисел в тестах
func SimulationWithRandom(enabled bool, delay time.Duration, errorRate float64, random RandomFloat) echo.MiddlewareFunc {
	// При выключенной симуляции возвращаем middleware без дополнительной работы на запросе
	if !enabled {
		return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Сначала добавляем задержку и прекращаем ожидание при отмене запроса клиентом
			if delay > 0 {
				timer := time.NewTimer(delay)
				defer timer.Stop()
				select {
				case <-timer.C:
				case <-c.Request().Context().Done():
					return c.Request().Context().Err()
				}
			}
			// После задержки разыгрываем искусственный отказ с настроенной вероятностью
			if errorRate > 0 && random() < errorRate {
				return apierrors.New(http.StatusServiceUnavailable, "simulated_failure", "service temporarily unavailable", nil)
			}
			return next(c)
		}
	}
}

// secureRandomFloat получает конкурентно-безопасное случайное число от нуля до единицы
func secureRandomFloat() float64 {
	var bytes [8]byte
	if _, err := cryptorand.Read(bytes[:]); err != nil {
		return 1
	}
	return float64(binary.LittleEndian.Uint64(bytes[:])) / float64(math.MaxUint64)
}
