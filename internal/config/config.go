package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultAuthSessionTTL = 24 * time.Hour

	defaultOneBotForwardWSPath = "/"
	defaultOneBotReverseWSPath = "/onebot/ws"

	OneBotModeDisabled  = "disabled"
	OneBotModeHTTP      = "http"
	OneBotModeHTTPS     = "https"
	OneBotModeWS        = "ws"
	OneBotModeWSS       = "wss"
	OneBotModeReverseWS = "reverse_ws"
)

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

// OneBotConfig 描述 OneBot v11 连接配置，用于向绑定的 QQ 号推送通知。
type OneBotConfig struct {
	// Mode 是连接方式开关：disabled、http、https、ws、wss、reverse_ws。
	Mode string `yaml:"mode"`
	// Host 和 Port 是 OneBot 服务地址；reverse_ws 模式下为监听地址，Host 可留空。
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	// WSPath 是 WebSocket 路径；ws/wss 默认 "/"，reverse_ws 默认 "/onebot/ws"。
	WSPath string `yaml:"ws_path"`
	// AccessToken 用于 HTTP 与 WebSocket 鉴权。
	AccessToken string `yaml:"access_token"`

	// URL 是 http/https/ws/wss 模式拼接出的完整地址。
	URL string `yaml:"-"`
	// ReverseAddr 和 ReversePath 是 reverse_ws 模式的监听地址与路径。
	ReverseAddr string `yaml:"-"`
	ReversePath string `yaml:"-"`
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
		OneBot: OneBotConfig{
			Mode: OneBotModeDisabled,
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
	if oneBotMode := strings.TrimSpace(os.Getenv("BILIGO_ONEBOT_MODE")); oneBotMode != "" {
		cfg.OneBot.Mode = oneBotMode
	}
	if oneBotHost := strings.TrimSpace(os.Getenv("BILIGO_ONEBOT_HOST")); oneBotHost != "" {
		cfg.OneBot.Host = oneBotHost
	}
	if oneBotPort := strings.TrimSpace(os.Getenv("BILIGO_ONEBOT_PORT")); oneBotPort != "" {
		parsed, err := strconv.Atoi(oneBotPort)
		if err != nil {
			return Config{}, fmt.Errorf("BILIGO_ONEBOT_PORT 必须是整数")
		}
		cfg.OneBot.Port = parsed
	}
	if oneBotWSPath := strings.TrimSpace(os.Getenv("BILIGO_ONEBOT_WS_PATH")); oneBotWSPath != "" {
		cfg.OneBot.WSPath = oneBotWSPath
	}
	if oneBotToken := strings.TrimSpace(os.Getenv("BILIGO_ONEBOT_TOKEN")); oneBotToken != "" {
		cfg.OneBot.AccessToken = oneBotToken
	}
	cfg.OneBot.AccessToken = strings.TrimSpace(cfg.OneBot.AccessToken)
	cfg.OneBot.Host = strings.Trim(strings.TrimSpace(cfg.OneBot.Host), "[]")
	cfg.OneBot.WSPath = strings.TrimSpace(cfg.OneBot.WSPath)

	mode := normalizeOneBotMode(cfg.OneBot.Mode)
	if mode == "" {
		return Config{}, fmt.Errorf("onebot.mode 必须是 disabled、http、https、ws、wss 或 reverse_ws")
	}
	cfg.OneBot.Mode = mode
	if mode != OneBotModeDisabled {
		if cfg.OneBot.Port <= 0 || cfg.OneBot.Port > 65535 {
			return Config{}, fmt.Errorf("onebot.port 必须在 1-65535 之间")
		}
		if _, _, err := net.SplitHostPort(cfg.OneBot.Host); err == nil {
			return Config{}, fmt.Errorf("onebot.host 不应包含端口，请使用 onebot.port")
		}
		if strings.Contains(cfg.OneBot.Host, "/") {
			return Config{}, fmt.Errorf("onebot.host 只填写主机名或 IP，不要包含协议或路径")
		}
		if mode != OneBotModeReverseWS && cfg.OneBot.Host == "" {
			return Config{}, fmt.Errorf("onebot.host 不能为空（%s 模式）", mode)
		}
		address := net.JoinHostPort(cfg.OneBot.Host, strconv.Itoa(cfg.OneBot.Port))
		switch mode {
		case OneBotModeHTTP:
			cfg.OneBot.URL = "http://" + address
		case OneBotModeHTTPS:
			cfg.OneBot.URL = "https://" + address
		case OneBotModeWS:
			cfg.OneBot.WSPath = normalizeOneBotWSPath(cfg.OneBot.WSPath, false)
			cfg.OneBot.URL = "ws://" + address + strings.TrimRight(cfg.OneBot.WSPath, "/")
		case OneBotModeWSS:
			cfg.OneBot.WSPath = normalizeOneBotWSPath(cfg.OneBot.WSPath, false)
			cfg.OneBot.URL = "wss://" + address + strings.TrimRight(cfg.OneBot.WSPath, "/")
		case OneBotModeReverseWS:
			cfg.OneBot.WSPath = normalizeOneBotWSPath(cfg.OneBot.WSPath, true)
			cfg.OneBot.ReverseAddr = address
			cfg.OneBot.ReversePath = cfg.OneBot.WSPath
		}
	}

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

func normalizeOneBotMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "off", "none", "disabled":
		return OneBotModeDisabled
	case OneBotModeHTTP:
		return OneBotModeHTTP
	case OneBotModeHTTPS:
		return OneBotModeHTTPS
	case "ws", "forward", "forward_ws":
		return OneBotModeWS
	case OneBotModeWSS:
		return OneBotModeWSS
	case "reverse", "reverse_ws":
		return OneBotModeReverseWS
	default:
		return ""
	}
}

func normalizeOneBotWSPath(raw string, reverse bool) string {
	path := strings.TrimSpace(raw)
	if path == "" {
		if reverse {
			return defaultOneBotReverseWSPath
		}
		return defaultOneBotForwardWSPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
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
