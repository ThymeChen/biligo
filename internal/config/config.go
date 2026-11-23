package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultAuthSessionTTL = 24 * time.Hour

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Logging  LoggingConfig  `yaml:"logging"`
	OneBot   OneBotConfig   `yaml:"onebot"`
	Path     string         `yaml:"-"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// AuthConfig 描述登录鉴权相关配置。
type AuthConfig struct {
	// SessionTTL 是扫码登录后签发的 Bearer Token 有效期，使用 Go duration 格式（如 30m、24h）。
	SessionTTL time.Duration `yaml:"session_ttl"`
}

// OneBotConfig 描述 OneBot v11 HTTP API 服务端的连接信息，用于向绑定的 QQ 号推送通知。
type OneBotConfig struct {
	APIURL      string `yaml:"api_url"`
	AccessToken string `yaml:"access_token"`
}

type LoggingConfig struct {
	Levels LogLevels         `yaml:"levels"`
	Color  string            `yaml:"color"`
	File   LoggingFileConfig `yaml:"file"`
}

type LoggingFileConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type LogLevels []string

func Load(path string) (Config, error) {
	cfg := Config{
		Server: ServerConfig{
			Addr: ":8080",
		},
		Database: DatabaseConfig{
			Path: "data/biligo.db",
		},
		Auth: AuthConfig{
			SessionTTL: defaultAuthSessionTTL,
		},
		Logging: LoggingConfig{
			Levels: LogLevels{"info", "warn", "error"},
			Color:  "auto",
			File: LoggingFileConfig{
				Path: "logs/biligo.log",
			},
		},
	}

	if path == "" {
		path = os.Getenv("BILIGO_CONFIG")
	}
	if path == "" {
		path = "config.yaml"
	}
	cfg.Path = path

	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Config{}, err
		}
	} else if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "data/biligo.db"
	}
	cfg.Logging.Color = normalizeLogColor(cfg.Logging.Color)
	if strings.TrimSpace(cfg.Logging.File.Path) == "" {
		cfg.Logging.File.Path = "logs/biligo.log"
	}
	if addr := os.Getenv("BILIGO_ADDR"); addr != "" {
		cfg.Server.Addr = addr
	}
	if dbPath := os.Getenv("BILIGO_DB"); dbPath != "" {
		cfg.Database.Path = dbPath
	}
	if levels := strings.TrimSpace(os.Getenv("BILIGO_LOG_LEVELS")); levels != "" {
		cfg.Logging.Levels = parseLogLevels(levels)
	}
	if color := strings.TrimSpace(os.Getenv("BILIGO_LOG_COLOR")); color != "" {
		cfg.Logging.Color = normalizeLogColor(color)
	}
	if oneBotURL := strings.TrimSpace(os.Getenv("BILIGO_ONEBOT_URL")); oneBotURL != "" {
		cfg.OneBot.APIURL = oneBotURL
	}
	if oneBotToken := strings.TrimSpace(os.Getenv("BILIGO_ONEBOT_TOKEN")); oneBotToken != "" {
		cfg.OneBot.AccessToken = oneBotToken
	}
	cfg.OneBot.APIURL = strings.TrimRight(strings.TrimSpace(cfg.OneBot.APIURL), "/")
	cfg.OneBot.AccessToken = strings.TrimSpace(cfg.OneBot.AccessToken)

	if ttl := strings.TrimSpace(os.Getenv("BILIGO_AUTH_TTL")); ttl != "" {
		parsed, err := time.ParseDuration(ttl)
		if err != nil {
			return Config{}, fmt.Errorf("BILIGO_AUTH_TTL 不是有效的时长：%w", err)
		}
		cfg.Auth.SessionTTL = parsed
	}
	if cfg.Auth.SessionTTL <= 0 {
		cfg.Auth.SessionTTL = defaultAuthSessionTTL
	}

	return cfg, nil
}

func (levels *LogLevels) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*levels = parseLogLevels(value.Value)
	case yaml.SequenceNode:
		items := make([]string, 0, len(value.Content))
		for _, item := range value.Content {
			items = append(items, strings.TrimSpace(item.Value))
		}
		*levels = items
	}
	return nil
}

func parseLogLevels(raw string) LogLevels {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	})
	levels := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			levels = append(levels, part)
		}
	}
	return levels
}

func normalizeLogColor(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "auto":
		return "auto"
	case "always", "on", "true", "force":
		return "always"
	case "never", "off", "false", "none":
		return "never"
	default:
		return "auto"
	}
}
