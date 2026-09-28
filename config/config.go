package config

import (
	"encoding/json"
	"os"

	"log/slog"
)

const DefaultReferenzZinssatzUrl = "https://www.bwo.admin.ch/de/entwicklung-referenzzinssatz-und-durchschnittszinssatz"

type Config struct {
	ReferenzZinssatzUrl        string
	DatabasePath               string
	DatabaseName               string
	Domain                     string
	Port                       string
	Ssl                        bool
	SMTP_Username              string
	SMTP_Password              string
	SMTP_Host                  string
	SMTP_Port                  string
	ReCaptchaSiteKey           string
	RecaptchaGoogleCloudApiKey string
	ReCaptchaProjectID         string
	RecaptchaMinScore          float32
	TrustedProxies             []string
	Debug                      bool
	Loki                       LokiConfig
}

// LokiConfig enables shipping logs to Grafana Loki; an empty Url turns it off.
type LokiConfig struct {
	// Url is the Loki address, e.g. http://loki.example.ch:3100; the push path is added if missing.
	Url string
	// Username and Password are sent as basic auth, as Grafana Cloud expects.
	Username string
	Password string
	// TenantId is sent as X-Scope-OrgID for multi-tenant Loki setups.
	TenantId string
	// Labels are added to every stream, next to service_name.
	Labels map[string]string
}

// BaseUrl is the public address used in links that are sent by mail.
func (c Config) BaseUrl() string {
	if c.Ssl {
		return "https://" + c.Domain
	}
	return "http://" + c.Domain
}

func LoadConfig(configPath string, logger *slog.Logger) Config {
	logger.Info("Loading config", "path", configPath)

	configFile, err := os.ReadFile(configPath)
	if err != nil {
		logger.Error("Failed to open config", "path", configPath, "error", err)
		os.Exit(1)
	}

	// The file holds secrets, so neither it nor its content ends up in the log.
	var config Config
	if err := json.Unmarshal(configFile, &config); err != nil {
		logger.Error("Failed to parse config", "path", configPath, "error", err)
		os.Exit(1)
	}

	if config.ReferenzZinssatzUrl == "" {
		config.ReferenzZinssatzUrl = DefaultReferenzZinssatzUrl
	}

	logger.Info("Config loaded", "baseUrl", config.BaseUrl(), "port", config.Port)
	return config
}
