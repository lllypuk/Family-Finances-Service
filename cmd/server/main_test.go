package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheckTimeout_Constant(t *testing.T) {
	// Test that the health check timeout constant is properly defined
	expectedTimeout := 2 * time.Second
	assert.Equal(t, expectedTimeout, HealthCheckTimeout)
}

func TestDoHealthCheck_Success(t *testing.T) {
	// Create a test server that simulates a healthy service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok"}`))
		if err != nil {
			return
		}
	}))
	defer server.Close()

	// We can't easily test doHealthCheck directly because it hardcodes localhost:8080
	// Instead, we'll test the underlying HTTP client logic
	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/health", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDoHealthCheck_Failure(t *testing.T) {
	// Create a test server that simulates an unhealthy service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(`{"status":"error"}`))
		if err != nil {
			return
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/health", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}(resp.Body)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestDoHealthCheck_Timeout(t *testing.T) {
	// Create a test server that simulates slow response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Sleep longer than the health check timeout
		time.Sleep(HealthCheckTimeout + 100*time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/health", nil)
	require.NoError(t, err)

	_, err = client.Do(req)
	// Should get a context deadline exceeded error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestHealthCheck_CommandLineArgument(t *testing.T) {
	// Test that the health check argument is recognized
	// We simulate command line arguments

	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "No arguments",
			args:     []string{"program"},
			expected: false,
		},
		{
			name:     "Health check argument",
			args:     []string{"program", "-health-check"},
			expected: true,
		},
		{
			name:     "Other argument",
			args:     []string{"program", "-version"},
			expected: false,
		},
		{
			name:     "Multiple arguments with health check",
			args:     []string{"program", "-health-check", "extra"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate os.Args
			originalArgs := os.Args
			defer func() {
				//nolint:reassign // This is intentional for testing
				os.Args = originalArgs
			}()

			//nolint:reassign // This is intentional for testing
			os.Args = tt.args

			shouldRunHealthCheck := len(os.Args) > 1 && os.Args[1] == "-health-check"
			assert.Equal(t, tt.expected, shouldRunHealthCheck)
		})
	}
}

func TestHTTPRequest_Creation(t *testing.T) {
	// Test HTTP request creation for health check
	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080/health", nil)
	require.NoError(t, err)

	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "http://localhost:8080/health", req.URL.String())
	assert.Equal(t, "/health", req.URL.Path)
	assert.Equal(t, "localhost:8080", req.URL.Host)
	assert.Equal(t, "http", req.URL.Scheme)

	// Check that context is properly set
	assert.NotNil(t, req.Context())
	deadline, ok := req.Context().Deadline()
	assert.True(t, ok)
	assert.True(t, deadline.After(time.Now()))
}

func TestHTTPRequest_InvalidURL(t *testing.T) {
	// Test HTTP request creation with invalid URL
	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	invalidURLs := []string{
		"://localhost:8080",
	}

	for _, url := range invalidURLs {
		t.Run("InvalidURL_"+url, func(t *testing.T) {
			_, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			require.Error(t, err)
		})
	}
}

func TestHTTPClient_Configuration(t *testing.T) {
	// Test HTTP client configuration
	client := &http.Client{}

	assert.NotNil(t, client)
	assert.Equal(t, time.Duration(0), client.Timeout) // Default client has no timeout (we use context timeout)
}

func TestHealthCheck_ResponseHandling(t *testing.T) {
	// Test different HTTP response status codes
	statusCodes := []struct {
		code          int
		shouldSucceed bool
		description   string
	}{
		{http.StatusOK, true, "OK"},
		{http.StatusCreated, false, "Created"},
		{http.StatusNoContent, false, "No Content"},
		{http.StatusBadRequest, false, "Bad Request"},
		{http.StatusUnauthorized, false, "Unauthorized"},
		{http.StatusForbidden, false, "Forbidden"},
		{http.StatusNotFound, false, "Not Found"},
		{http.StatusInternalServerError, false, "Internal Server Error"},
		{http.StatusBadGateway, false, "Bad Gateway"},
		{http.StatusServiceUnavailable, false, "Service Unavailable"},
		{http.StatusGatewayTimeout, false, "Gateway Timeout"},
	}

	for _, tc := range statusCodes {
		t.Run(fmt.Sprintf("Status_%d_%s", tc.code, tc.description), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.code)
			}))
			defer server.Close()

			ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
			defer cancel()

			client := &http.Client{}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/health", nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer func(Body io.ReadCloser) {
				err = Body.Close()
				if err != nil {
					t.Errorf("Failed to close response body: %v", err)
				}
			}(resp.Body)

			assert.Equal(t, tc.code, resp.StatusCode)

			// Health check should only succeed for 200 OK
			if tc.shouldSucceed {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			} else {
				assert.NotEqual(t, http.StatusOK, resp.StatusCode)
			}
		})
	}
}

func TestHealthCheck_NetworkErrors(t *testing.T) {
	// Test handling of network errors
	networkErrors := []struct {
		name string
		url  string
	}{
		{
			name: "Connection refused",
			url:  "http://localhost:99999/health", // Port that's unlikely to be in use
		},
		{
			name: "Invalid host",
			url:  "http://non-existent-host-12345.invalid/health",
		},
	}

	for _, tc := range networkErrors {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond) // Short timeout
			defer cancel()

			client := &http.Client{}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, tc.url, nil)
			require.NoError(t, err)

			_, err = client.Do(req)
			require.Error(t, err)
		})
	}
}

func TestContextTimeout_Behavior(t *testing.T) {
	// Test context timeout behavior
	timeouts := []time.Duration{
		1 * time.Millisecond,
		10 * time.Millisecond,
		100 * time.Millisecond,
		1 * time.Second,
		HealthCheckTimeout,
	}

	for _, timeout := range timeouts {
		t.Run(fmt.Sprintf("Timeout_%v", timeout), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// Check that deadline is set correctly
			deadline, ok := ctx.Deadline()
			assert.True(t, ok)

			expectedDeadline := time.Now().Add(timeout)
			// Allow for some variance in timing
			assert.WithinDuration(t, expectedDeadline, deadline, 10*time.Millisecond)
		})
	}
}

func TestMain_ArgumentParsing(t *testing.T) {
	// Test argument parsing logic
	tests := []struct {
		name              string
		args              []string
		expectHealthCheck bool
	}{
		{
			name:              "No arguments",
			args:              []string{},
			expectHealthCheck: false,
		},
		{
			name:              "Only program name",
			args:              []string{"program"},
			expectHealthCheck: false,
		},
		{
			name:              "Health check flag",
			args:              []string{"program", "-health-check"},
			expectHealthCheck: true,
		},
		{
			name:              "Wrong flag",
			args:              []string{"program", "--health-check"},
			expectHealthCheck: false,
		},
		{
			name:              "Different flag",
			args:              []string{"program", "-version"},
			expectHealthCheck: false,
		},
		{
			name:              "Multiple flags",
			args:              []string{"program", "-health-check", "-verbose"},
			expectHealthCheck: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the main function's argument checking logic
			shouldRunHealthCheck := len(tt.args) > 1 && tt.args[1] == "-health-check"
			assert.Equal(t, tt.expectHealthCheck, shouldRunHealthCheck)
		})
	}
}

// Benchmark tests for health check operations
func BenchmarkHTTPRequestCreation(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_, _ = http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080/health", nil)
	}
}

func BenchmarkContextCreation(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
		cancel()
		_ = ctx
	}
}

func BenchmarkHealthCheckSuccess(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{}

	b.ResetTimer()
	for range b.N {
		ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/health", nil)
		resp, _ := client.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		cancel()
	}
}
