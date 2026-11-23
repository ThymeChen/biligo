package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestLoadAuthSessionTTL(t *testing.T) {
	t.Setenv("BILIGO_CONFIG", "")
	t.Setenv("BILIGO_AUTH_TTL", "")

	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Auth.SessionTTL != 24*time.Hour {
		t.Fatalf("Auth.SessionTTL = %s, want 24h", cfg.Auth.SessionTTL)
	}

	if err := os.WriteFile(path, []byte("auth:\n  session_ttl: 30m\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load with file ttl: %v", err)
	}
	if cfg.Auth.SessionTTL != 30*time.Minute {
		t.Fatalf("Auth.SessionTTL = %s, want 30m", cfg.Auth.SessionTTL)
	}

	t.Setenv("BILIGO_AUTH_TTL", "2h")
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load with env ttl: %v", err)
	}
	if cfg.Auth.SessionTTL != 2*time.Hour {
		t.Fatalf("Auth.SessionTTL = %s, want 2h", cfg.Auth.SessionTTL)
	}

	t.Setenv("BILIGO_AUTH_TTL", "not-a-duration")
	if _, err := Load(path); err == nil {
		t.Fatal("Load with invalid env ttl should fail")
	}
}

func TestLoadOneBotConfig(t *testing.T) {
	t.Setenv("BILIGO_CONFIG", "")
	t.Setenv("BILIGO_ONEBOT_MODE", "")
	t.Setenv("BILIGO_ONEBOT_HOST", "")
	t.Setenv("BILIGO_ONEBOT_PORT", "")
	t.Setenv("BILIGO_ONEBOT_WS_PATH", "")

	path := filepath.Join(t.TempDir(), "config.yaml")
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load defaults: %v", err)
	}
	if cfg.OneBot.Mode != OneBotModeDisabled {
		t.Fatalf("OneBot.Mode = %q, want disabled", cfg.OneBot.Mode)
	}

	write(`
onebot:
  mode: "http"
  host: "127.0.0.1"
  port: 3000
  access_token: "secret"
`)
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load http mode: %v", err)
	}
	if cfg.OneBot.Mode != OneBotModeHTTP || cfg.OneBot.URL != "http://127.0.0.1:3000" || cfg.OneBot.AccessToken != "secret" {
		t.Fatalf("unexpected http mode config: %#v", cfg.OneBot)
	}

	write(`
onebot:
  mode: "https"
  host: "onebot.example.com"
  port: 443
`)
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load https mode: %v", err)
	}
	if cfg.OneBot.Mode != OneBotModeHTTPS || cfg.OneBot.URL != "https://onebot.example.com:443" {
		t.Fatalf("unexpected https mode config: %#v", cfg.OneBot)
	}

	write(`
onebot:
  mode: "ws"
  host: "127.0.0.1"
  port: 3001
`)
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load ws mode: %v", err)
	}
	if cfg.OneBot.Mode != OneBotModeWS || cfg.OneBot.URL != "ws://127.0.0.1:3001" {
		t.Fatalf("unexpected ws mode config: %#v", cfg.OneBot)
	}
	if cfg.OneBot.WSPath != "/" {
		t.Fatalf("OneBot.WSPath = %q, want /", cfg.OneBot.WSPath)
	}

	write(`
onebot:
  mode: "ws"
  host: "127.0.0.1"
  port: 3001
  ws_path: "onebot/v11"
`)
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load ws custom path: %v", err)
	}
	if cfg.OneBot.WSPath != "/onebot/v11" || cfg.OneBot.URL != "ws://127.0.0.1:3001/onebot/v11" {
		t.Fatalf("unexpected ws custom path config: %#v", cfg.OneBot)
	}

	write(`
onebot:
  mode: "wss"
  host: "example.com"
  port: 443
`)
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load wss mode: %v", err)
	}
	if cfg.OneBot.Mode != OneBotModeWSS || cfg.OneBot.URL != "wss://example.com:443" {
		t.Fatalf("unexpected wss mode config: %#v", cfg.OneBot)
	}

	write(`
onebot:
  mode: "reverse_ws"
  host: ""
  port: 8090
`)
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load reverse ws mode: %v", err)
	}
	if cfg.OneBot.Mode != OneBotModeReverseWS || cfg.OneBot.ReverseAddr != ":8090" || cfg.OneBot.ReversePath != "/onebot/ws" {
		t.Fatalf("unexpected reverse ws config: %#v", cfg.OneBot)
	}

	write(`
onebot:
  mode: "reverse_ws"
  host: "127.0.0.1"
  port: 8091
`)
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load reverse ws with host: %v", err)
	}
	if cfg.OneBot.ReverseAddr != "127.0.0.1:8091" || cfg.OneBot.ReversePath != "/onebot/ws" {
		t.Fatalf("unexpected reverse ws host config: %#v", cfg.OneBot)
	}

	write(`
onebot:
  mode: "reverse_ws"
  host: "127.0.0.1"
  port: 8092
  ws_path: "custom/ws"
`)
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load reverse ws custom path: %v", err)
	}
	if cfg.OneBot.ReverseAddr != "127.0.0.1:8092" || cfg.OneBot.ReversePath != "/custom/ws" || cfg.OneBot.WSPath != "/custom/ws" {
		t.Fatalf("unexpected reverse ws custom path: %#v", cfg.OneBot)
	}

	t.Setenv("BILIGO_ONEBOT_MODE", "wss")
	t.Setenv("BILIGO_ONEBOT_HOST", "env.example.com")
	t.Setenv("BILIGO_ONEBOT_PORT", "9443")
	t.Setenv("BILIGO_ONEBOT_WS_PATH", "/env/ws")
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load env override: %v", err)
	}
	if cfg.OneBot.Mode != OneBotModeWSS || cfg.OneBot.URL != "wss://env.example.com:9443/env/ws" {
		t.Fatalf("unexpected env override: %#v", cfg.OneBot)
	}

	t.Setenv("BILIGO_ONEBOT_MODE", "")
	t.Setenv("BILIGO_ONEBOT_HOST", "")
	t.Setenv("BILIGO_ONEBOT_PORT", "")
	t.Setenv("BILIGO_ONEBOT_WS_PATH", "")
	write("onebot:\n  mode: \"unknown\"\n  port: 3000\n")
	if _, err := Load(path); err == nil {
		t.Fatal("Load with unknown mode should fail")
	}

	write("onebot:\n  mode: \"http\"\n  host: \"127.0.0.1\"\n  port: 0\n")
	if _, err := Load(path); err == nil {
		t.Fatal("Load with invalid port should fail")
	}

	write("onebot:\n  mode: \"http\"\n  host: \"127.0.0.1:3000\"\n  port: 3000\n")
	if _, err := Load(path); err == nil {
		t.Fatal("Load with port in host should fail")
	}

	write("onebot:\n  mode: \"http\"\n  host: \"http://127.0.0.1\"\n  port: 3000\n")
	if _, err := Load(path); err == nil {
		t.Fatal("Load with scheme in host should fail")
	}

	write("onebot:\n  mode: \"http\"\n  host: \"\"\n  port: 3000\n")
	if _, err := Load(path); err == nil {
		t.Fatal("Load http without host should fail")
	}

	write("onebot:\n  mode: \"ws\"\n  host: \"127.0.0.1\"\n  port: 70000\n")
	if _, err := Load(path); err == nil {
		t.Fatal("Load with out-of-range port should fail")
	}

	t.Setenv("BILIGO_ONEBOT_PORT", "not-a-port")
	write("onebot:\n  mode: \"http\"\n  host: \"127.0.0.1\"\n  port: 3000\n")
	if _, err := Load(path); err == nil {
		t.Fatal("Load with invalid env port should fail")
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
