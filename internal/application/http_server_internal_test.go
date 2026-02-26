package application

import (
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestHTTPServer_BuildNetHTTPServer_UsesConfiguredTimeouts(t *testing.T) {
	e := echo.New()
	s := &HTTPServer{
		echo: e,
		config: &Config{
			Host:         "127.0.0.1",
			Port:         "8080",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 7 * time.Second,
			IdleTimeout:  11 * time.Second,
		},
	}

	server := s.buildNetHTTPServer("127.0.0.1:8080")

	assert.Equal(t, "127.0.0.1:8080", server.Addr)
	assert.Equal(t, 5*time.Second, server.ReadTimeout)
	assert.Equal(t, 7*time.Second, server.WriteTimeout)
	assert.Equal(t, 11*time.Second, server.IdleTimeout)
	assert.Same(t, e, server.Handler)
}
