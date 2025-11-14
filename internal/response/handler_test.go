package response_test

import (
	"net/http"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/lexfrei/go-unifi/internal/response"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockResponse is a test double for API responses.
type mockResponse struct {
	statusCode int
}

func (m *mockResponse) StatusCode() int {
	return m.statusCode
}

// mockData is a test type for response data.
type mockData struct {
	Value string
}

func TestHandle(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}
		data := &mockData{Value: "test"}

		result, err := response.Handle(resp, data, nil, "test error")
		require.NoError(t, err, "Handle() should not return error")

		assert.Same(t, data, result, "Handle() result mismatch")
	})

	t.Run("client error", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}
		data := &mockData{Value: "test"}
		clientErr := errors.New("network error")

		_, err := response.Handle(resp, data, clientErr, "test error")
		require.Error(t, err, "Handle() should return error")

		assert.ErrorIs(t, err, clientErr, "Handle() error should wrap client error")
	})

	t.Run("wrong status code", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusNotFound}
		data := &mockData{Value: "test"}

		_, err := response.Handle(resp, data, nil, "test error")
		require.Error(t, err, "Handle() should return error")
	})

	t.Run("nil data", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}

		_, err := response.Handle[mockData](resp, nil, nil, "test error")
		require.Error(t, err, "Handle() should return error")
	})
}

func TestHandleWithStatus(t *testing.T) {
	t.Parallel()

	t.Run("success with custom status", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusCreated}
		data := &mockData{Value: "test"}

		result, err := response.HandleWithStatus(resp, data, nil, "test error", http.StatusCreated)
		require.NoError(t, err, "HandleWithStatus() should not return error")

		assert.Same(t, data, result, "HandleWithStatus() result mismatch")
	})

	t.Run("wrong status code", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}
		data := &mockData{Value: "test"}

		_, err := response.HandleWithStatus(resp, data, nil, "test error", http.StatusCreated)
		require.Error(t, err, "HandleWithStatus() should return error")
	})
}

func TestHandleNoContent(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}

		err := response.HandleNoContent(resp, nil, "test error")
		require.NoError(t, err, "HandleNoContent() should not return error")
	})

	t.Run("client error", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}
		clientErr := errors.New("network error")

		err := response.HandleNoContent(resp, clientErr, "test error")
		require.Error(t, err, "HandleNoContent() should return error")

		assert.ErrorIs(t, err, clientErr, "HandleNoContent() error should wrap client error")
	})

	t.Run("wrong status code", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusNotFound}

		err := response.HandleNoContent(resp, nil, "test error")
		require.Error(t, err, "HandleNoContent() should return error")
	})
}

func TestHandleNoContentWithStatus(t *testing.T) {
	t.Parallel()

	t.Run("success with custom status", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusNoContent}

		err := response.HandleNoContentWithStatus(resp, nil, "test error", http.StatusNoContent)
		require.NoError(t, err, "HandleNoContentWithStatus() should not return error")
	})

	t.Run("wrong status code", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}

		err := response.HandleNoContentWithStatus(resp, nil, "test error", http.StatusNoContent)
		require.Error(t, err, "HandleNoContentWithStatus() should return error")
	})
}
