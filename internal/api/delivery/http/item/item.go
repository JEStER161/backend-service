package item

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"backend-service/delivery/http/model"
	"backend-service/internal/api/domain"
	"backend-service/internal/api/models"
	apierrors "backend-service/pkg/api/errors"
)

// Handler предоставляет HTTP-операции для работы с записями
type Handler struct {
	item domain.Item
}

// NewHandler создаёт HTTP-обработчик операций с записями
func NewHandler(item domain.Item) *Handler {
	return &Handler{item: item}
}

// validateCreateRequest нормализует имя и проверяет обязательные поля и длины
func validateCreateRequest(request model.CreateItemRequest) (string, string, error) {
	if request.Name == nil {
		return "", "", apierrors.New(http.StatusBadRequest, "validation_error", "field name is required", nil)
	}
	name := strings.TrimSpace(*request.Name)
	nameLength := utf8.RuneCountInString(name)
	if nameLength < 1 || nameLength > 200 {
		return "", "", apierrors.New(http.StatusBadRequest, "validation_error", "field name must contain between 1 and 200 characters", nil)
	}
	if request.Value == nil || *request.Value == "" {
		return "", "", apierrors.New(http.StatusBadRequest, "validation_error", "field value is required", nil)
	}
	if utf8.RuneCountInString(*request.Value) > 5000 {
		return "", "", apierrors.New(http.StatusBadRequest, "validation_error", "field value must not exceed 5000 characters", nil)
	}
	return name, *request.Value, nil
}

// ensureJSONEOF запрещает дополнительные JSON-значения после тела запроса
func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected value after JSON object")
		}
		return err
	}
	return nil
}

// toResponse преобразует внутреннюю модель в публичный HTTP-ответ
func toResponse(value models.Item) model.Item {
	return model.Item{
		ID:        value.ID.String(),
		Name:      value.Name,
		Value:     value.Value,
		CreatedAt: value.CreatedAt,
	}
}
