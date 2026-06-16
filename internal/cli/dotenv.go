package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// LoadDotEnv loads KEY=VALUE pairs from the file at path into the process
// environment. It is intentionally minimal (no command/variable expansion):
// blank lines and lines beginning with '#' are ignored, an optional leading
// "export " is stripped, and a single or double quoted value is unquoted.
//
// Existing environment variables are never overwritten, so real environment
// values (including those injected by docker-compose's env_file) take
// precedence over the file. A missing file is not an error.
func LoadDotEnv(path string) error {
	f, err := os.Open(path) //nolint:gosec // path is operator-provided configuration
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: missing '=' in %q", path, lineNo, line)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("%s:%d: empty key", path, lineNo)
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		if err := os.Setenv(key, unquote(strings.TrimSpace(value))); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// unquote removes a single pair of matching surrounding quotes, if present.
func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}

	return s
}
