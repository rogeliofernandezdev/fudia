package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("# comment\nFOODS_DOTENV_TEST=ready\nFOODS_QUOTED=\"value\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("FOODS_DOTENV_TEST"); _ = os.Unsetenv("FOODS_QUOTED") })
	if err := LoadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("FOODS_DOTENV_TEST") != "ready" || os.Getenv("FOODS_QUOTED") != "value" {
		t.Fatal("variables were not loaded")
	}
}
