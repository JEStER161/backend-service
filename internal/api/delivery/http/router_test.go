package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend-service/configs"
	"backend-service/delivery/http/model"
	httpdelivery "backend-service/internal/api/delivery/http"
	"backend-service/internal/api/delivery/http/health"
	itemhttp "backend-service/internal/api/delivery/http/item"
	"backend-service/internal/api/domain"
	"backend-service/internal/api/models"

	"github.com/google/uuid"
)

type fakeItem struct {
	create func(context.Context, string, string) (models.Item, error)
	get    func(context.Context, uuid.UUID) (models.Item, error)
}

// Create передаёт вызов тестовой функции создания записи
func (f fakeItem) Create(ctx context.Context, name, value string) (models.Item, error) {
	if f.create == nil {
		panic("unexpected Create call")
	}
	return f.create(ctx, name, value)
}

// GetByID передаёт вызов тестовой функции чтения записи
func (f fakeItem) GetByID(ctx context.Context, id uuid.UUID) (models.Item, error) {
	if f.get == nil {
		panic("unexpected GetByID call")
	}
	return f.get(ctx, id)
}

type fakeChecker struct{ err error }

// PingContext возвращает настроенный результат тестовой проверки БД
func (f fakeChecker) PingContext(context.Context) error { return f.err }

// testRouter собирает HTTP-маршрутизатор с тестовыми зависимостями
func testRouter(itemDomain fakeItem, checker fakeChecker, simulation configs.Simulation) http.Handler {
	config := &configs.Config{
		Logging: configs.Logging{RequestIDFieldName: "x-request-id"},
		Listen: configs.Listen{HTTP: configs.HTTP{
			MaxRequestProcessingTime: configs.Duration{Duration: time.Second},
		}},
		Application: configs.Application{InstanceID: "test-instance", Simulation: simulation},
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return httpdelivery.NewRouter(
		logger,
		config,
		itemhttp.NewHandler(itemDomain),
		health.NewHandler(checker, config.Application.InstanceID, time.Second),
	).Handler()
}

// TestCreateItem проверяет успешное создание записи через HTTP
func TestCreateItem(t *testing.T) {
	id := uuid.New()
	createdAt := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	router := testRouter(fakeItem{
		create: func(_ context.Context, name, value string) (models.Item, error) {
			if name != "test item" || value != "test value" {
				t.Fatalf("Create() got name=%q value=%q", name, value)
			}
			return models.Item{ID: id, Name: name, Value: value, CreatedAt: createdAt}, nil
		},
	}, fakeChecker{}, configs.Simulation{})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/items", strings.NewReader(`{"name":"  test item  ","value":"test value"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Instance-ID") != "test-instance" {
		t.Fatal("missing instance header")
	}
	var body model.Item
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID != id.String() || !body.CreatedAt.Equal(createdAt) {
		t.Fatalf("body = %#v", body)
	}
}

// TestCreateItemValidation проверяет варианты невалидного тела запроса
func TestCreateItemValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ""},
		{name: "invalid JSON", body: "{"},
		{name: "unknown field", body: `{"name":"item","value":"value","extra":true}`},
		{name: "missing name", body: `{"value":"value"}`},
		{name: "blank name", body: `{"name":"   ","value":"value"}`},
		{name: "missing value", body: `{"name":"item"}`},
		{name: "empty value", body: `{"name":"item","value":""}`},
		{name: "second value", body: `{"name":"item","value":"value"} {}`},
		{name: "long name", body: `{"name":"` + strings.Repeat("я", 201) + `","value":"value"}`},
		{name: "long value", body: `{"name":"item","value":"` + strings.Repeat("я", 5001) + `"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := testRouter(fakeItem{}, fakeChecker{}, configs.Simulation{})
			request := httptest.NewRequest(http.MethodPost, "/api/v1/items", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var body model.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Error.Code != "validation_error" {
				t.Fatalf("error code = %q", body.Error.Code)
			}
		})
	}
}

// TestGetItemNotFound проверяет преобразование доменной ошибки в HTTP 404
func TestGetItemNotFound(t *testing.T) {
	router := testRouter(fakeItem{
		get: func(context.Context, uuid.UUID) (models.Item, error) {
			return models.Item{}, domain.ErrItemNotFound
		},
	}, fakeChecker{}, configs.Simulation{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/items/"+uuid.NewString(), nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

// TestGetItemInvalidUUID проверяет отклонение некорректного UUID
func TestGetItemInvalidUUID(t *testing.T) {
	router := testRouter(fakeItem{}, fakeChecker{}, configs.Simulation{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/items/not-a-uuid", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

// TestHealth проверяет ответы при доступной и недоступной PostgreSQL
func TestHealth(t *testing.T) {
	tests := []struct {
		name       string
		checker    fakeChecker
		wantStatus int
		wantState  string
	}{
		{name: "available", checker: fakeChecker{}, wantStatus: http.StatusOK, wantState: "ok"},
		{name: "unavailable", checker: fakeChecker{err: errors.New("database unavailable")}, wantStatus: http.StatusServiceUnavailable, wantState: "unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := testRouter(fakeItem{}, test.checker, configs.Simulation{})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var body model.Health
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Status != test.wantState || body.Database != test.wantState || body.InstanceID != "test-instance" {
				t.Fatalf("body = %#v", body)
			}
		})
	}
}

// TestSimulationDoesNotAffectHealth проверяет изоляцию health check от симуляции
func TestSimulationDoesNotAffectHealth(t *testing.T) {
	simulation := configs.Simulation{IsEnabled: true, ErrorRate: 1}
	itemDomain := fakeItem{
		get: func(context.Context, uuid.UUID) (models.Item, error) {
			t.Fatal("simulation should reject API request before domain")
			return models.Item{}, nil
		},
	}
	router := testRouter(itemDomain, fakeChecker{}, simulation)

	healthResponse := httptest.NewRecorder()
	router.ServeHTTP(healthResponse, httptest.NewRequest(http.MethodGet, "/health", nil))
	if healthResponse.Code != http.StatusOK {
		t.Fatalf("health status = %d", healthResponse.Code)
	}

	apiResponse := httptest.NewRecorder()
	router.ServeHTTP(apiResponse, httptest.NewRequest(http.MethodGet, "/api/v1/items/"+uuid.NewString(), nil))
	if apiResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("API status = %d, body = %s", apiResponse.Code, apiResponse.Body.String())
	}
}
