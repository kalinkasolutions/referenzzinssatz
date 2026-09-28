package config

import (
	"encoding/json"
	"os"

	"github.com/kalinkasolutions/referenzzinssatz/logger"
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
}

// BaseUrl is the public address used in links that are sent by mail.
func (c Config) BaseUrl() string {
	if c.Ssl {
		return "https://" + c.Domain
	}
	return "http://" + c.Domain
}

func LoadConfig(configPath string, logger logger.ILogger) Config {
	logger.Info("Loading config from %s", configPath)

	configFile, err := os.ReadFile(configPath)
	if err != nil {
		logger.Error("Failed to open config at %s: %v", configPath, err)
		os.Exit(1)
	}

	// The file holds secrets, so neither it nor its content ends up in the log.
	var config Config
	if err := json.Unmarshal(configFile, &config); err != nil {
		logger.Error("Failed to parse config at %s: %v", configPath, err)
		os.Exit(1)
	}

	if config.ReferenzZinssatzUrl == "" {
		config.ReferenzZinssatzUrl = DefaultReferenzZinssatzUrl
	}

	logger.Info("Config loaded, serving %s on port %s", config.BaseUrl(), config.Port)
	return config
}
