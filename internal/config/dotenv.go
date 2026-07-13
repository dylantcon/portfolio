package config

import (
	"bufio"
	"os"
	"strings"
)

// LoadDotEnv reads a .env file (one KEY=VALUE per line) from the given path and
// sets any variables that aren't already present in the environment. Variables
// already set in the real environment (e.g. via systemd or the shell) always
// win, so this is safe to call unconditionally. A missing file is not an error.
func LoadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // no .env file is fine
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
