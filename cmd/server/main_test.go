package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func newResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

func TestHealthCheckTimeout_Constant(t *testing.T) {
	assert.Equal(t, 2*time.Second, HealthCheckTimeout)
}

func TestBuildHealthCheckURL(t *testing.T) {
	tests := []struct {
		name string
		host string
		port string
		want string
	}{
		{
			name: "defaults",
			want: "http://localhost:8080/health",
		},
		{
			name: "custom host and port",
			host: "127.0.0.1",
			port: "9090",
			want: "http://127.0.0.1:9090/health",
		},
		{
			name: "wildcard ipv4 host maps to localhost",
			host: "0.0.0.0",
			port: "8081",
			want: "http://localhost:8081/health",
		},
		{
			name: "wildcard ipv6 host maps to localhost",
			host: "::",
			port: "8082",
			want: "http://localhost:8082/health",
		},
		{
			name: "ipv6 host is bracketed in URL",
			host: "::1",
			port: "8080",
			want: "http://[::1]:8080/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, buildHealthCheckURL(tt.host, tt.port))
		})
	}
}

func TestCheckHealth_Success(t *testing.T) {
	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "/health", req.URL.Path)
		assert.Equal(t, "127.0.0.1:8080", req.URL.Host)
		return newResponse(http.StatusOK), nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	err := checkHealth(ctx, client, "http://127.0.0.1:8080/health")
	require.NoError(t, err)
}

func TestCheckHealth_NonOKStatus(t *testing.T) {
	client := newTestClient(func(_ *http.Request) (*http.Response, error) {
		return newResponse(http.StatusServiceUnavailable), nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	err := checkHealth(ctx, client, "http://localhost:8080/health")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code")
}

func TestCheckHealth_ClientError(t *testing.T) {
	client := newTestClient(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})

	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	err := checkHealth(ctx, client, "http://localhost:8080/health")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestCheckHealth_Timeout(t *testing.T) {
	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := checkHealth(ctx, client, "http://localhost:8080/health")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestCheckHealth_InvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	err := checkHealth(ctx, newTestClient(func(_ *http.Request) (*http.Response, error) {
		t.Fatal("transport should not be called for invalid URL")
		return nil, errors.New("unexpected transport call")
	}), "://bad-url")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestDoHealthCheckWithURL_ReturnCodes(t *testing.T) {
	successClient := newTestClient(func(_ *http.Request) (*http.Response, error) {
		return newResponse(http.StatusOK), nil
	})
	assert.Equal(t, 0, doHealthCheckWithURL(successClient, "http://localhost:8080/health"))

	failClient := newTestClient(func(_ *http.Request) (*http.Response, error) {
		return newResponse(http.StatusInternalServerError), nil
	})
	assert.Equal(t, 1, doHealthCheckWithURL(failClient, "http://localhost:8080/health"))
}

func TestHealthCheck_CommandLineArgument(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{name: "No arguments", args: []string{"program"}, expected: false},
		{name: "Health check argument", args: []string{"program", "-health-check"}, expected: true},
		{name: "Other argument", args: []string{"program", "-version"}, expected: false},
		{
			name:     "Multiple arguments with health check",
			args:     []string{"program", "-health-check", "extra"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalArgs := os.Args
			defer func() {
				//nolint:reassign // Intentional in tests
				os.Args = originalArgs
			}()

			//nolint:reassign // Intentional in tests
			os.Args = tt.args

			shouldRunHealthCheck := len(os.Args) > 1 && os.Args[1] == "-health-check"
			assert.Equal(t, tt.expected, shouldRunHealthCheck)
		})
	}
}
