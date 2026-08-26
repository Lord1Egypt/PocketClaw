package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/logger"
)

func TestLoggerSuppressesOnlyRoutineSuccessfulGatewayPolls(t *testing.T) {
	initialLevel := logger.GetLevel()
	logger.DisableFileLogging()
	logger.SetLevel(logger.DEBUG)
	logger.SetConsoleLevel(logger.DEBUG)
	logger.DisableConsole()
	t.Cleanup(func() {
		logger.DisableFileLogging()
		logger.EnableConsole()
		logger.SetConsoleLevel(initialLevel)
		logger.SetLevel(initialLevel)
	})

	logPath := filepath.Join(t.TempDir(), "http.log")
	if err := logger.EnableFileLogging(logPath); err != nil {
		t.Fatalf("EnableFileLogging: %v", err)
	}

	tests := []struct {
		name    string
		method  string
		path    string
		status  int
		emitted bool
	}{
		{name: "logs success", method: http.MethodGet, path: "/api/gateway/logs", status: http.StatusOK},
		{name: "status success", method: http.MethodGet, path: "/api/gateway/status", status: http.StatusOK},
		{name: "logs failure", method: http.MethodGet, path: "/api/gateway/logs", status: http.StatusInternalServerError, emitted: true},
		{name: "status failure", method: http.MethodGet, path: "/api/gateway/status", status: http.StatusInternalServerError, emitted: true},
		{name: "config patch", method: http.MethodPatch, path: "/api/config", status: http.StatusOK, emitted: true},
		{name: "models", method: http.MethodGet, path: "/api/models", status: http.StatusOK, emitted: true},
		{name: "unknown route", method: http.MethodGet, path: "/api/unknown", status: http.StatusNotFound, emitted: true},
		{name: "unexpected poll method", method: http.MethodPost, path: "/api/gateway/logs", status: http.StatusOK, emitted: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, nil)
			handler.ServeHTTP(recorder, request)
		})
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	output := string(raw)
	for _, tt := range tests {
		needle := tt.method + " " + tt.path + " " + strconv.Itoa(tt.status)
		if got := strings.Contains(output, needle); got != tt.emitted {
			t.Errorf("%s emitted = %v, want %v; logs:\n%s", tt.name, got, tt.emitted, output)
		}
	}
}
