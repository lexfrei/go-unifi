// Package response provides generic handlers for API responses to eliminate boilerplate.
package response

import (
	"net/http"

	"github.com/cockroachdb/errors"
)

// StatusCoder is an interface for response types that can return HTTP status code.
// All oapi-codegen generated response types implement this interface.
type StatusCoder interface {
	StatusCode() int
}

// Handle is a generic handler for API responses that return data (GET, POST, PUT).
// It checks for errors, validates status code (expects 200 OK), and ensures response data is non-nil.
//
// Usage:
//
//	resp, err := c.client.GetDeviceByIdWithResponse(ctx, siteID, deviceID)
//	return response.Handle(resp, resp.JSON200, err, "failed to get device")
func Handle[T any](resp StatusCoder, data *T, err error, errorMsg string) (*T, error) {
	return HandleWithStatus(resp, data, err, errorMsg, http.StatusOK)
}

// HandleWithStatus is like Handle but allows specifying the expected status code.
// Use this for endpoints that return non-200 success codes (e.g., 201 Created).
//
// Usage:
//
//	resp, err := c.client.CreateResourceWithResponse(ctx, req)
//	return response.HandleWithStatus(resp, resp.JSON201, err, "failed to create resource", http.StatusCreated)
func HandleWithStatus[T any](resp StatusCoder, data *T, err error, errorMsg string, expectedStatus int) (*T, error) {
	if err != nil {
		return nil, errors.Wrap(err, errorMsg)
	}

	if resp.StatusCode() != expectedStatus {
		//nolint:wrapcheck // Creating new error for non-expected status, no source error to wrap
		return nil, errors.Newf("API error: status=%d", resp.StatusCode())
	}

	if data == nil {
		return nil, errors.New("empty response from API")
	}

	return data, nil
}

// HandleNoContent is a handler for API responses that don't return data (DELETE).
// It checks for errors and validates status code (expects 200 OK).
//
// Usage:
//
//	resp, err := c.client.DeleteResourceWithResponse(ctx, id)
//	return response.HandleNoContent(resp, err, "failed to delete resource")
func HandleNoContent(resp StatusCoder, err error, errorMsg string) error {
	return HandleNoContentWithStatus(resp, err, errorMsg, http.StatusOK)
}

// HandleNoContentWithStatus is like HandleNoContent but allows specifying expected status code.
// Use this for DELETE endpoints that return 204 No Content.
//
// Usage:
//
//	resp, err := c.client.DeleteResourceWithResponse(ctx, id)
//	return response.HandleNoContentWithStatus(resp, err, "failed to delete resource", http.StatusNoContent)
func HandleNoContentWithStatus(resp StatusCoder, err error, errorMsg string, expectedStatus int) error {
	if err != nil {
		return errors.Wrap(err, errorMsg)
	}

	if resp.StatusCode() != expectedStatus {
		//nolint:wrapcheck // Creating new error for non-expected status, no source error to wrap
		return errors.Newf("API error: status=%d", resp.StatusCode())
	}

	return nil
}
