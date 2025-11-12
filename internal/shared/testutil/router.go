package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/validator"
	"github.com/gin-gonic/gin"
)

// SetupTestRouter creates a test Gin router without middleware
func SetupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	// Register custom validators for testing
	_ = validator.RegisterAll()

	return gin.New()
}

// MakeRequest is a helper to make HTTP requests in tests
type TestRequest struct {
	Method string
	URL    string
	Body   interface{}
}

// ExecuteRequest executes a test HTTP request and returns the response
func ExecuteRequest(t *testing.T, router *gin.Engine, req TestRequest) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq := httptest.NewRequest(req.Method, req.URL, bodyReader)
	httpReq.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpReq)

	return recorder
}

// ParseResponse parses the JSON response body into the given struct
func ParseResponse(t *testing.T, recorder *httptest.ResponseRecorder, v interface{}) {
	t.Helper()

	if err := json.Unmarshal(recorder.Body.Bytes(), v); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}
}
