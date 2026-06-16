package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/drewstreib/exporter-unifi-protect/internal/cli"
)

func writeEnv(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	return path
}

func TestLoadDotEnv(t *testing.T) {
	path := writeEnv(t, `
# a comment
export UNIFI_HOST=https://nvr.example
UNIFI_USERNAME='quoted-user'
UNIFI_PASSWORD="p@ss=word"
UNIFI_WEB_LISTEN_ADDRESSES=:7777
`)

	for _, k := range []string{"UNIFI_HOST", "UNIFI_USERNAME", "UNIFI_PASSWORD", "UNIFI_WEB_LISTEN_ADDRESSES"} {
		os.Unsetenv(k)
	}

	if err := cli.LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	want := map[string]string{
		"UNIFI_HOST":                 "https://nvr.example",
		"UNIFI_USERNAME":             "quoted-user", // single quotes stripped
		"UNIFI_PASSWORD":             "p@ss=word",   // only first '=' splits; quotes stripped
		"UNIFI_WEB_LISTEN_ADDRESSES": ":7777",
	}
	for k, v := range want {
		if got := os.Getenv(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
		os.Unsetenv(k)
	}
}

func TestLoadDotEnvDoesNotOverride(t *testing.T) {
	t.Setenv("UNIFI_HOST", "https://real-env")

	path := writeEnv(t, "UNIFI_HOST=https://from-file\n")
	if err := cli.LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv: %v", err)
	}

	if got := os.Getenv("UNIFI_HOST"); got != "https://real-env" {
		t.Errorf("real environment overridden: got %q", got)
	}
}

func TestLoadDotEnvMissingFileIsOK(t *testing.T) {
	if err := cli.LoadDotEnv(filepath.Join(t.TempDir(), "does-not-exist")); err != nil {
		t.Errorf("missing file should not error, got %v", err)
	}
}

func TestLoadDotEnvMalformed(t *testing.T) {
	path := writeEnv(t, "this-line-has-no-equals\n")
	if err := cli.LoadDotEnv(path); err == nil {
		t.Error("expected an error for a line without '='")
	}
}
