package auth_test

import (
	"net/http"
	"testing"

	"github.com/changhyeonkim/pray-together/go-api-server/internal/auth"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/member"
	sharedError "github.com/changhyeonkim/pray-together/go-api-server/internal/shared/error"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestEnvironment creates all dependencies needed for auth handler tests
func setupTestEnvironment(t *testing.T) (*auth.AuthHandler, *testutil.MockTokenManager) {
	t.Helper()

	// Setup test database
	db := testutil.SetupTestDB(t)
	t.Cleanup(func() {
		testutil.CleanupTestDB(t, db)
	})

	// Setup dependencies
	memberRepo := member.NewMemberRepository()
	mockTokenManager := testutil.NewMockTokenManager()
	authService := auth.NewAuthService(db, memberRepo, mockTokenManager)
	authHandler := auth.NewAuthHandler(authService)

	return authHandler, mockTokenManager
}

func TestSignup_Success(t *testing.T) {
	// Given: Setup test environment
	authHandler, _ := setupTestEnvironment(t)

	router := testutil.SetupTestRouter()
	router.POST("/api/v1/auth/signup", authHandler.Signup)

	// Given: Valid signup request
	request := testutil.TestRequest{
		Method: http.MethodPost,
		URL:    "/api/v1/auth/signup",
		Body: auth.SignupRequest{
			Name:        "Test User",
			Email:       "test@example.com",
			PhoneNumber: "010-1234-5678",
			Password:    "password123",
		},
	}

	// When: Execute signup request
	recorder := testutil.ExecuteRequest(t, router, request)

	// Then: Verify response
	assert.Equal(t, http.StatusCreated, recorder.Code)
}

func TestSignup_DuplicateEmail(t *testing.T) {
	// Given: Setup test environment
	authHandler, _ := setupTestEnvironment(t)

	router := testutil.SetupTestRouter()
	router.POST("/api/v1/auth/signup", authHandler.Signup)

	// Given: Create first user
	firstRequest := testutil.TestRequest{
		Method: http.MethodPost,
		URL:    "/api/v1/auth/signup",
		Body: auth.SignupRequest{
			Name:        "Test User",
			Email:       "duplicate@example.com",
			PhoneNumber: "010-1234-5678",
			Password:    "password123",
		},
	}

	firstRecorder := testutil.ExecuteRequest(t, router, firstRequest)
	require.Equal(t, http.StatusCreated, firstRecorder.Code)

	// When: Try to create another user with same email
	duplicateRequest := testutil.TestRequest{
		Method: http.MethodPost,
		URL:    "/api/v1/auth/signup",
		Body: auth.SignupRequest{
			Name:        "Another User",
			Email:       "duplicate@example.com", // Same email
			PhoneNumber: "010-9876-5432",
			Password:    "password456",
		},
	}

	duplicateRecorder := testutil.ExecuteRequest(t, router, duplicateRequest)

	// Then: Verify error response
	assert.Equal(t, http.StatusConflict, duplicateRecorder.Code)

	var errorResponse sharedError.ErrorResponse
	testutil.ParseResponse(t, duplicateRecorder, &errorResponse)
	assert.NotEmpty(t, errorResponse.Status)
	assert.NotEmpty(t, errorResponse.Message)
	assert.Equal(t, "MEMBER-002", errorResponse.Code)
}

func TestSignup_ValidationError_MissingRequiredFields(t *testing.T) {
	// Given: Setup test environment
	authHandler, _ := setupTestEnvironment(t)

	router := testutil.SetupTestRouter()
	router.POST("/api/v1/auth/signup", authHandler.Signup)

	testCases := []struct {
		name        string
		requestBody map[string]string
		description string
	}{
		{
			name: "Missing name",
			requestBody: map[string]string{
				"email":       "test@example.com",
				"phoneNumber": "010-1234-5678",
				"password":    "password123",
			},
			description: "Should fail when name is missing",
		},
		{
			name: "Missing email",
			requestBody: map[string]string{
				"name":        "Test User",
				"phoneNumber": "010-1234-5678",
				"password":    "password123",
			},
			description: "Should fail when email is missing",
		},
		{
			name: "Missing phoneNumber",
			requestBody: map[string]string{
				"name":     "Test User",
				"email":    "test@example.com",
				"password": "password123",
			},
			description: "Should fail when phoneNumber is missing",
		},
		{
			name: "Missing password",
			requestBody: map[string]string{
				"name":        "Test User",
				"email":       "test@example.com",
				"phoneNumber": "010-1234-5678",
			},
			description: "Should fail when password is missing",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// When: Execute request with missing field
			request := testutil.TestRequest{
				Method: http.MethodPost,
				URL:    "/api/v1/auth/signup",
				Body:   tc.requestBody,
			}

			recorder := testutil.ExecuteRequest(t, router, request)

			// Then: Verify validation error
			assert.Equal(t, http.StatusBadRequest, recorder.Code, tc.description)

			var errorResponse sharedError.ErrorResponse
			testutil.ParseResponse(t, recorder, &errorResponse)
			assert.NotEmpty(t, errorResponse.Status, tc.description)
			assert.NotEmpty(t, errorResponse.Message, tc.description)
			assert.NotEmpty(t, errorResponse.Code, tc.description)
		})
	}
}

func TestSignup_ValidationError_InvalidEmail(t *testing.T) {
	// Given: Setup test environment
	authHandler, _ := setupTestEnvironment(t)

	router := testutil.SetupTestRouter()
	router.POST("/api/v1/auth/signup", authHandler.Signup)

	// Given: Request with invalid email format
	request := testutil.TestRequest{
		Method: http.MethodPost,
		URL:    "/api/v1/auth/signup",
		Body: auth.SignupRequest{
			Name:        "Test User",
			Email:       "invalid-email-format", // Invalid email
			PhoneNumber: "010-1234-5678",
			Password:    "password123",
		},
	}

	// When: Execute request
	recorder := testutil.ExecuteRequest(t, router, request)

	// Then: Verify validation error
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var errorResponse sharedError.ErrorResponse
	testutil.ParseResponse(t, recorder, &errorResponse)
	assert.NotEmpty(t, errorResponse.Message)
}

func TestSignup_ValidationError_PasswordTooShort(t *testing.T) {
	// Given: Setup test environment
	authHandler, _ := setupTestEnvironment(t)

	router := testutil.SetupTestRouter()
	router.POST("/api/v1/auth/signup", authHandler.Signup)

	// Given: Request with short password
	request := testutil.TestRequest{
		Method: http.MethodPost,
		URL:    "/api/v1/auth/signup",
		Body: auth.SignupRequest{
			Name:        "Test User",
			Email:       "test@example.com",
			PhoneNumber: "010-1234-5678",
			Password:    "short", // Less than 8 characters
		},
	}

	// When: Execute request
	recorder := testutil.ExecuteRequest(t, router, request)

	// Then: Verify validation error
	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var errorResponse sharedError.ErrorResponse
	testutil.ParseResponse(t, recorder, &errorResponse)
	assert.NotEmpty(t, errorResponse.Message)
}
