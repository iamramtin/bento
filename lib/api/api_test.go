package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jeffail/benthos/v3/lib/log"
	"github.com/Jeffail/benthos/v3/lib/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIEnableCORS(t *testing.T) {
	conf := NewConfig()
	conf.CORS.Enabled = true
	conf.CORS.AllowedOrigins = []string{"*"}

	s, err := New("", "", conf, nil, log.Noop(), metrics.Noop())
	require.NoError(t, err)

	handler := s.server.Handler

	request, _ := http.NewRequest("OPTIONS", "/version", nil)
	request.Header.Add("Origin", "meow")
	request.Header.Add("Access-Control-Request-Method", "POST")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "*", response.Header().Get("Access-Control-Allow-Origin"))
}

func TestAPIEnableCORSOrigins(t *testing.T) {
	conf := NewConfig()
	conf.CORS.Enabled = true
	conf.CORS.AllowedOrigins = []string{"foo", "bar"}

	s, err := New("", "", conf, nil, log.Noop(), metrics.Noop())
	require.NoError(t, err)

	handler := s.server.Handler

	request, _ := http.NewRequest("OPTIONS", "/version", nil)
	request.Header.Add("Origin", "foo")
	request.Header.Add("Access-Control-Request-Method", "POST")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "foo", response.Header().Get("Access-Control-Allow-Origin"))

	request, _ = http.NewRequest("OPTIONS", "/version", nil)
	request.Header.Add("Origin", "bar")
	request.Header.Add("Access-Control-Request-Method", "POST")

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "bar", response.Header().Get("Access-Control-Allow-Origin"))

	request, _ = http.NewRequest("OPTIONS", "/version", nil)
	request.Header.Add("Origin", "baz")
	request.Header.Add("Access-Control-Request-Method", "POST")

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "", response.Header().Get("Access-Control-Allow-Origin"))
}

func TestAPIEnableCORSNoHeaders(t *testing.T) {
	conf := NewConfig()
	conf.CORS.Enabled = true

	_, err := New("", "", conf, nil, log.Noop(), metrics.Noop())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify at least one allowed origin")
}
