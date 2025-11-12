package response_test

import (
	"net/http"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/lexfrei/go-unifi/internal/response"
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
		if err != nil {
			t.Fatalf("Handle() error = %v, want nil", err)
		}

		if result != data {
			t.Errorf("Handle() result = %v, want %v", result, data)
		}
	})

	t.Run("client error", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}
		data := &mockData{Value: "test"}
		clientErr := errors.New("network error")

		_, err := response.Handle(resp, data, clientErr, "test error")
		if err == nil {
			t.Fatal("Handle() error = nil, want error")
		}

		if !errors.Is(err, clientErr) {
			t.Errorf("Handle() error should wrap client error")
		}
	})

	t.Run("wrong status code", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusNotFound}
		data := &mockData{Value: "test"}

		_, err := response.Handle(resp, data, nil, "test error")
		if err == nil {
			t.Fatal("Handle() error = nil, want error")
		}
	})

	t.Run("nil data", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}

		_, err := response.Handle[mockData](resp, nil, nil, "test error")
		if err == nil {
			t.Fatal("Handle() error = nil, want error")
		}
	})
}

func TestHandleWithStatus(t *testing.T) {
	t.Parallel()

	t.Run("success with custom status", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusCreated}
		data := &mockData{Value: "test"}

		result, err := response.HandleWithStatus(resp, data, nil, "test error", http.StatusCreated)
		if err != nil {
			t.Fatalf("HandleWithStatus() error = %v, want nil", err)
		}

		if result != data {
			t.Errorf("HandleWithStatus() result = %v, want %v", result, data)
		}
	})

	t.Run("wrong status code", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}
		data := &mockData{Value: "test"}

		_, err := response.HandleWithStatus(resp, data, nil, "test error", http.StatusCreated)
		if err == nil {
			t.Fatal("HandleWithStatus() error = nil, want error")
		}
	})
}

func TestHandleNoContent(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}

		err := response.HandleNoContent(resp, nil, "test error")
		if err != nil {
			t.Fatalf("HandleNoContent() error = %v, want nil", err)
		}
	})

	t.Run("client error", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}
		clientErr := errors.New("network error")

		err := response.HandleNoContent(resp, clientErr, "test error")
		if err == nil {
			t.Fatal("HandleNoContent() error = nil, want error")
		}

		if !errors.Is(err, clientErr) {
			t.Errorf("HandleNoContent() error should wrap client error")
		}
	})

	t.Run("wrong status code", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusNotFound}

		err := response.HandleNoContent(resp, nil, "test error")
		if err == nil {
			t.Fatal("HandleNoContent() error = nil, want error")
		}
	})
}

func TestHandleNoContentWithStatus(t *testing.T) {
	t.Parallel()

	t.Run("success with custom status", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusNoContent}

		err := response.HandleNoContentWithStatus(resp, nil, "test error", http.StatusNoContent)
		if err != nil {
			t.Fatalf("HandleNoContentWithStatus() error = %v, want nil", err)
		}
	})

	t.Run("wrong status code", func(t *testing.T) {
		t.Parallel()

		resp := &mockResponse{statusCode: http.StatusOK}

		err := response.HandleNoContentWithStatus(resp, nil, "test error", http.StatusNoContent)
		if err == nil {
			t.Fatal("HandleNoContentWithStatus() error = nil, want error")
		}
	})
}
