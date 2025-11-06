package config

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"net/url"
	"os"
	"sync"
)

const defaultConfigPath = "./config.yml"

type AppConfig struct {
	LogLevel uint8 `yaml:"log_level"`
	Debug    bool  `yaml:"debug"`
	Testing  bool  `yaml:"testing"`
	Database struct {
		Debug    bool   `yaml:"debug"`
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Name     string `yaml:"name"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"database"`
	Server struct {
		CookieDomain string   `yaml:"cookie_domain"`
		Host         string   `yaml:"host"`
		AllowedHosts []string `yaml:"allowed_hosts"`
		Paths        struct {
			Internal string `yaml:"internal"`
			Media    string `yaml:"media"`
		} `yaml:"paths"`
	} `yaml:"server"`
	Email struct {
		HookEmail string `yaml:"hook_email"`
		Default   struct {
			From     string `yaml:"from"`
			Host     string `yaml:"host"`
			Port     string `yaml:"port"`
			User     string `yaml:"user"`
			Password string `yaml:"password"`
		} `yaml:"default"`
	} `yaml:"email"`
	Sentry struct {
		DSN string `yaml:"dsn"`
	} `yaml:"sentry"`
	hostUrl   *url.URL
	wsHostUrl *url.URL
}

var (
	once sync.Once
	cfg  *AppConfig
)

// Singleton для конфигурации
func App() *AppConfig {
	once.Do(func() {
		cfg = prepareConfig()
	})
	return cfg
}

// Получение пути к конфигурационному файлу
func getConfigPath() string {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}
	if os.Getenv("TESTING") == "true" {
		return "./testing.config.yml"
	}
	return defaultConfigPath
}

// Загрузка и обработка конфигурации
func prepareConfig() *AppConfig {
	configPath := getConfigPath()
	log.Info().Str("path", configPath).Msg("Using config path")

	// Проверка наличия файла
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatal().Err(err).Str("path", configPath).Msg("Config file not found")
	}

	buffer, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading config file")
	}

	var c AppConfig
	if err := yaml.Unmarshal(buffer, &c); err != nil {
		log.Fatal().Err(err).Msg("Error parsing YAML config")
	}

	// Валидация конфигурации
	validateConfig(&c)

	// Разбираем URL сервера
	hostUrl, err := url.Parse(c.Server.Host)
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid server host URL")
	}
	c.hostUrl = hostUrl

	// Определяем WebSocket URL
	wsHostUrl := *hostUrl
	switch wsHostUrl.Scheme {
	case "https":
		wsHostUrl.Scheme = "wss"
	case "http":
		wsHostUrl.Scheme = "ws"
	default:
		log.Fatal().Interface("url", wsHostUrl).Msg("Invalid WebSocket host")
	}
	c.wsHostUrl = &wsHostUrl

	log.Info().Msg("Configuration successfully loaded")
	return &c
}

// Валидация необходимых полей
func validateConfig(c *AppConfig) {
	if c.Server.Host == "" {
		log.Fatal().Msg("Server host is required")
	}
	if c.Database.Host == "" || c.Database.Port == "" || c.Database.Name == "" ||
		c.Database.User == "" || c.Database.Password == "" {
		log.Fatal().Msg("Database configuration is incomplete")
	}
}

func (c *AppConfig) HostURL() *url.URL {
	return c.hostUrl
}

func (c *AppConfig) WSHostURL() *url.URL {
	return c.wsHostUrl
}

func (c *AppConfig) PGUri() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.Database.User, c.Database.Password, c.Database.Host, c.Database.Port, c.Database.Name)
}
