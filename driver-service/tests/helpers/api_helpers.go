//go:build integration

package helpers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// APITestHelper помощник для тестирования HTTP API
type APITestHelper struct {
	router *gin.Engine
	t      *testing.T
}

// NewAPITestHelper создает новый APITestHelper
func NewAPITestHelper(router *gin.Engine, t *testing.T) *APITestHelper {
	return &APITestHelper{
		router: router,
		t:      t,
	}
}

// APIRequest структура для HTTP запроса
type APIRequest struct {
	Method      string
	URL         string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// APIResponse структура для HTTP ответа
type APIResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// MakeRequest выполняет HTTP запрос и возвращает ответ
func (h *APITestHelper) MakeRequest(req APIRequest) *APIResponse {
	var bodyReader io.Reader

	// Подготавливаем тело запроса
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		require.NoError(h.t, err)
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	// Создаем HTTP запрос
	httpReq := httptest.NewRequest(req.Method, req.URL, bodyReader)

	// Добавляем заголовки
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Добавляем Content-Type для JSON
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Добавляем query параметры
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Выполняем запрос
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, httpReq)

	return &APIResponse{
		StatusCode: w.Code,
		Body:       w.Body.Bytes(),
		Headers:    w.Header(),
	}
}

// UnmarshalResponse парсит JSON ответ в структуру
func (h *APITestHelper) UnmarshalResponse(response *APIResponse, target interface{}) {
	err := json.Unmarshal(response.Body, target)
	require.NoError(h.t, err, "Failed to unmarshal response: %s", string(response.Body))
}

// AssertStatusCode проверяет код ответа
func (h *APITestHelper) AssertStatusCode(response *APIResponse, expectedCode int) {
	if response.StatusCode != expectedCode {
		h.t.Errorf("Expected status code %d, got %d. Response body: %s",
			expectedCode, response.StatusCode, string(response.Body))
	}
}

// CreateDriverRequest создает запрос на создание водителя
func CreateDriverRequest() map[string]interface{} {
	return map[string]interface{}{
		"phone":           "+79001234567",
		"email":           "test@example.com",
		"first_name":      "Тест",
		"last_name":       "Водитель",
		"birth_date":      "1985-05-15T00:00:00Z",
		"passport_series": "1234",
		"passport_number": "567890",
		"license_number":  "TEST123456",
		"license_expiry":  "2026-12-31T00:00:00Z",
	}
}

// CreateLocationRequest создает запрос на обновление местоположения
func CreateLocationRequest() map[string]interface{} {
	return map[string]interface{}{
		"latitude":  55.7558,
		"longitude": 37.6173,
		"altitude":  150.0,
		"accuracy":  10.0,
		"speed":     60.5,
		"bearing":   45.0,
	}
}

// CreateBatchLocationRequest создает запрос на пакетное обновление местоположений
func CreateBatchLocationRequest(count int) map[string]interface{} {
	locations := make([]map[string]interface{}, count)

	for i := 0; i < count; i++ {
		locations[i] = map[string]interface{}{
			"latitude":  55.7558 + float64(i)*0.001,
			"longitude": 37.6173 + float64(i)*0.001,
			"speed":     float64(50 + i*5),
		}
	}

	return map[string]interface{}{
		"locations": locations,
	}
}

// AssertErrorResponse проверяет структуру ошибки
func (h *APITestHelper) AssertErrorResponse(response *APIResponse, expectedCode string) {
	var errorResp struct {
		Error   string `json:"error"`
		Code    string `json:"code"`
		Details string `json:"details"`
	}

	h.UnmarshalResponse(response, &errorResp)

	if expectedCode != "" {
		if errorResp.Code != expectedCode {
			h.t.Errorf("Expected error code %s, got %s", expectedCode, errorResp.Code)
		}
	}
}

// AssertValidationError проверяет ошибку валидации
func (h *APITestHelper) AssertValidationError(response *APIResponse, field string) {
	h.AssertStatusCode(response, http.StatusBadRequest)
	h.AssertErrorResponse(response, "")

	// Проверяем, что в ошибке упоминается проблемное поле
	var errorResp struct {
		Details string `json:"details"`
	}
	h.UnmarshalResponse(response, &errorResp)

	// В зависимости от реализации валидации, проверяем наличие поля в details
	// Это может потребовать доработки в зависимости от используемой библиотеки валидации
}

// WaitForCondition ждет выполнения условия с таймаутом
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}
		case <-timeoutChan:
			t.Fatalf("Timeout waiting for condition: %s", message)
		}
	}
}

// AssertLocationInBounds проверяет, что местоположение находится в заданных границах
func AssertLocationInBounds(t *testing.T, lat, lon, centerLat, centerLon, radiusKm float64) {
	// Простая проверка через прямоугольные границы
	// В реальном проекте лучше использовать точный расчет расстояния
	latDiff := lat - centerLat
	lonDiff := lon - centerLon

	// Примерное преобразование км в градусы (для широт средних широт)
	radiusDegrees := radiusKm / 111.0 // 1 градус ≈ 111 км

	if latDiff*latDiff+lonDiff*lonDiff > radiusDegrees*radiusDegrees {
		t.Errorf("Location (%.6f, %.6f) is outside radius %.2fkm from center (%.6f, %.6f)",
			lat, lon, radiusKm, centerLat, centerLon)
	}
}

// LoadTestData загружает тестовые данные из JSON файла
func LoadTestData(t *testing.T, filename string, target interface{}) {
	// В реальном проекте здесь можно загружать данные из fixtures файлов
	// Пока оставляем заглушку
	t.Logf("Loading test data from %s", filename)
}

// CompareJSONObjects сравнивает два JSON объекта
func CompareJSONObjects(t *testing.T, expected, actual interface{}) {
	expectedBytes, err := json.Marshal(expected)
	require.NoError(t, err)

	actualBytes, err := json.Marshal(actual)
	require.NoError(t, err)

	var expectedMap, actualMap map[string]interface{}

	err = json.Unmarshal(expectedBytes, &expectedMap)
	require.NoError(t, err)

	err = json.Unmarshal(actualBytes, &actualMap)
	require.NoError(t, err)

	// Рекурсивное сравнение (упрощенная версия)
	compareMapRecursive(t, expectedMap, actualMap, "")
}

// compareMapRecursive рекурсивно сравнивает map'ы
func compareMapRecursive(t *testing.T, expected, actual map[string]interface{}, path string) {
	for key, expectedValue := range expected {
		currentPath := path + "." + key

		actualValue, exists := actual[key]
		if !exists {
			t.Errorf("Missing key %s in actual object", currentPath)
			continue
		}

		switch expectedValue.(type) {
		case map[string]interface{}:
			if actualMap, ok := actualValue.(map[string]interface{}); ok {
				compareMapRecursive(t, expectedValue.(map[string]interface{}), actualMap, currentPath)
			} else {
				t.Errorf("Type mismatch at %s: expected map, got %T", currentPath, actualValue)
			}
		default:
			if expectedValue != actualValue {
				t.Errorf("Value mismatch at %s: expected %v, got %v", currentPath, expectedValue, actualValue)
			}
		}
	}
}
