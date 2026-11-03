package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaultsAndEnvOverrides(t *testing.T) {
	t.Setenv("BILIGO_CONFIG", "")
	t.Setenv("BILIGO_ADDR", ":9999")
	t.Setenv("BILIGO_DB", "env.db")
	t.Setenv("BILIGO_LOG_COLOR", "")

	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Addr != ":9999" || cfg.Database.Path != "env.db" {
		t.Fatalf("env overrides were not applied: %#v", cfg)
	}
	if cfg.Logging.Color != "auto" {
		t.Fatalf("Logging.Color = %q, want auto", cfg.Logging.Color)
	}
	if cfg.Logging.File.Enabled {
		t.Fatal("Logging.File.Enabled should default to false")
	}
	if cfg.Logging.File.Path != "logs/biligo.log" {
		t.Fatalf("Logging.File.Path = %q, want logs/biligo.log", cfg.Logging.File.Path)
	}
}

func TestLoadLogLevelsFromConfigAndEnv(t *testing.T) {
	t.Setenv("BILIGO_CONFIG", "")
	t.Setenv("BILIGO_LOG_LEVELS", "")
	t.Setenv("BILIGO_LOG_COLOR", "")

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
server:
  addr: ":8080"
database:
  path: "data/biligo.db"
logging:
  levels: none
  color: never
  file:
    enabled: true
    path: "logs/test.log"
`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := strings.Join(cfg.Logging.Levels, ","); got != "none" {
		t.Fatalf("Logging.Levels = %q, want none", got)
	}
	if cfg.Logging.Color != "never" {
		t.Fatalf("Logging.Color = %q, want never", cfg.Logging.Color)
	}
	if !cfg.Logging.File.Enabled || cfg.Logging.File.Path != "logs/test.log" {
		t.Fatalf("Logging.File = %#v, want enabled logs/test.log", cfg.Logging.File)
	}

	t.Setenv("BILIGO_LOG_LEVELS", "warn,error")
	t.Setenv("BILIGO_LOG_COLOR", "always")
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load with env: %v", err)
	}
	if got := strings.Join(cfg.Logging.Levels, ","); got != "warn,error" {
		t.Fatalf("Logging.Levels = %q, want warn,error", got)
	}
	if cfg.Logging.Color != "always" {
		t.Fatalf("Logging.Color = %q, want always", cfg.Logging.Color)
	}
}
