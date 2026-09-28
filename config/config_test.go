package config

import (
	"os"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/mocks"
)

func TestConfig(t *testing.T) {
	path := t.TempDir() + "/config.json"
	err := os.WriteFile(path, []byte(`{
    "ReferenzZinssatzUrl": "ReferenzZinssatzUrl",
    "DatabasePath": "DatabasePath",
    "DatabaseName": "DatabaseName",
    "Ssl": false,
    "Domain": "Domain",
    "Port": "Port",
    "SMTP_Username": "SMTP_Username",
    "SMTP_Password": "SMTP_Password",
    "SMTP_Host": "SMTP_Host",
    "SMTP_Port": "SMTP_Port",
    "ReCaptchaSiteKey": "ReCaptchaSiteKey",
    "RecaptchaGoogleCloudApiKey": "RecaptchaGoogleCloudApiKey",
    "ReCaptchaProjectID": "ReCaptchaProjectID",
    "RecaptchaMinScore": 0.7
}`), 0644)

	assert.Equal(t, nil, err)

	config := LoadConfig(path, mocks.NewLoggerMock())

	assert.Equal(t, "ReferenzZinssatzUrl", config.ReferenzZinssatzUrl)
	assert.Equal(t, "DatabasePath", config.DatabasePath)
	assert.Equal(t, "DatabaseName", config.DatabaseName)
	assert.Equal(t, false, config.Ssl)
	assert.Equal(t, "Domain", config.Domain)
	assert.Equal(t, "Port", config.Port)
	assert.Equal(t, "SMTP_Username", config.SMTP_Username)
	assert.Equal(t, "SMTP_Password", config.SMTP_Password)
	assert.Equal(t, "SMTP_Host", config.SMTP_Host)
	assert.Equal(t, "SMTP_Port", config.SMTP_Port)
	assert.Equal(t, "ReCaptchaSiteKey", config.ReCaptchaSiteKey)
	assert.Equal(t, "RecaptchaGoogleCloudApiKey", config.RecaptchaGoogleCloudApiKey)
	assert.Equal(t, "ReCaptchaProjectID", config.ReCaptchaProjectID)
	assert.Equal(t, float32(0.7), config.RecaptchaMinScore)
}

func TestConfigDefaultsSourceUrl(t *testing.T) {
	path := t.TempDir() + "/config.json"
	err := os.WriteFile(path, []byte(`{"Domain": "example.ch", "Ssl": true}`), 0644)
	assert.Equal(t, nil, err)

	config := LoadConfig(path, mocks.NewLoggerMock())

	assert.Equal(t, DefaultReferenzZinssatzUrl, config.ReferenzZinssatzUrl)
	assert.Equal(t, "https://example.ch", config.BaseUrl())
}
