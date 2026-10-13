package configurations

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name  string
		value *string
		want  int
	}{
		{name: "unset uses default", value: nil, want: 30},
		{name: "empty uses default", value: ptr(""), want: 30},
		{name: "explicit zero is kept", value: ptr("0"), want: 0},
		{name: "value is parsed", value: ptr("45"), want: 45},
		{name: "spaces are trimmed", value: ptr(" 7 "), want: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == nil {
				os.Unsetenv("TEST_INT")
			} else {
				t.Setenv("TEST_INT", *tt.value)
			}

			if got := GetEnvInt("TEST_INT", false, 30); got != tt.want {
				t.Errorf("GetEnvInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

// Invalid values call log.Fatalf, so they run in a subprocess
func TestGetEnvIntInvalidExits(t *testing.T) {
	if value := os.Getenv("TEST_INT_SUBPROCESS"); value != "" {
		os.Setenv("TEST_INT", value)
		GetEnvInt("TEST_INT", false, 30)
		return
	}

	tests := []struct{ value, wantMessage string }{
		{value: "-1", wantMessage: "must not be negative"},
		{value: "abc", wantMessage: "must be a valid integer"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestGetEnvIntInvalidExits$")
			cmd.Env = append(os.Environ(), "TEST_INT_SUBPROCESS="+tt.value)
			output, err := cmd.CombinedOutput()

			if err == nil {
				t.Fatalf("expected exit with error for %q", tt.value)
			}
			if !strings.Contains(string(output), tt.wantMessage) {
				t.Errorf("output %q does not contain %q", output, tt.wantMessage)
			}
		})
	}
}

func TestGetEnvFileMode(t *testing.T) {
	os.Unsetenv("TEST_MODE")
	if got := GetEnvFileMode("TEST_MODE", 0o644); got != 0o644 {
		t.Errorf("unset: got %o, want 644", got)
	}

	t.Setenv("TEST_MODE", "0600")
	if got := GetEnvFileMode("TEST_MODE", 0o644); got != 0o600 {
		t.Errorf("0600: got %o, want 600", got)
	}
}

func TestGetPostgresEnvConfigUsesPrefix(t *testing.T) {
	t.Setenv("ANALYTICS_DB_DSN", "host=analytics")
	t.Setenv("ANALYTICS_DB_MAX_CONN", "5")
	t.Setenv("ANALYTICS_DB_AUTO_MIGRATE", "true")

	got := GetPostgresEnvConfig("analytics", "ANALYTICS_DB_")

	if got.Name != "analytics" || got.DSN != "host=analytics" || got.MaxConn != 5 || !got.AutoMigrate {
		t.Errorf("unexpected config: %+v", got)
	}
	if got.MaxConnIdle != 10 {
		t.Errorf("unset key should use its default, got MaxConnIdle=%d", got.MaxConnIdle)
	}
}

func ptr(s string) *string { return &s }

// Headers the frontend sends on every request must pass the browser's CORS preflight
func TestCORSAllowsFrontendHeaders(t *testing.T) {
	t.Setenv("PORT", "3000")
	t.Setenv("CLIENT_ID", "TEST")

	allowed := GetENVConfig().Application.CORSConfig.AllowHeaders

	for _, header := range []string{"X-Client-Id", "X-Client-Version", "X-Renew-Token", "Authorization", "Content-Type", "X-Request-Id"} {
		if !slices.Contains(allowed, header) {
			t.Errorf("CORS AllowHeaders is missing %q", header)
		}
	}
}
