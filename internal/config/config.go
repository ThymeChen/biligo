package config

import (
	"errors"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
	Path     string         `yaml:"-"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
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
