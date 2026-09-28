package recaptcha

import (
	"context"
	"fmt"
	"os"
	"time"

	recaptcha "cloud.google.com/go/recaptchaenterprise/v2/apiv1"
	recaptchapb "cloud.google.com/go/recaptchaenterprise/v2/apiv1/recaptchaenterprisepb"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"google.golang.org/api/option"
	"log/slog"
)

type IRecaptcha interface {
	CreateAssessment(token string, recaptchaAction string) bool
}

type Recaptcha struct {
	logger *slog.Logger
	config config.Config
	client *recaptcha.Client
}

func NewReCaptcha(logger *slog.Logger, config config.Config) *Recaptcha {
	client, err := recaptcha.NewClient(context.Background(), option.WithAPIKey(config.RecaptchaGoogleCloudApiKey))
	if err != nil {
		logger.Error("Failed to create reCAPTCHA client", "error", err)
		os.Exit(1)
	}

	return &Recaptcha{
		logger: logger,
		config: config,
		client: client,
	}
}

// CreateAssessment reports whether the token is valid, was issued for the
// expected action and scores at least the configured minimum.
func (r *Recaptcha) CreateAssessment(token string, recaptchaAction string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := r.client.CreateAssessment(ctx, &recaptchapb.CreateAssessmentRequest{
		Parent: fmt.Sprintf("projects/%s", r.config.ReCaptchaProjectID),
		Assessment: &recaptchapb.Assessment{
			Event: &recaptchapb.Event{
				Token:   token,
				SiteKey: r.config.ReCaptchaSiteKey,
			},
		},
	})
	if err != nil {
		r.logger.Error("Failed to create reCAPTCHA assessment", "error", err)
		return false
	}

	tokenProperties := response.GetTokenProperties()
	if !tokenProperties.GetValid() {
		r.logger.Warn("reCAPTCHA token invalid", "reason", tokenProperties.GetInvalidReason().String())
		return false
	}
	if tokenProperties.GetAction() != recaptchaAction {
		r.logger.Warn("reCAPTCHA action does not match", "action", tokenProperties.GetAction(), "expected", recaptchaAction)
		return false
	}

	score := response.GetRiskAnalysis().GetScore()
	if score < r.config.RecaptchaMinScore {
		r.logger.Warn("reCAPTCHA score below minimum", "score", score, "minimum", r.config.RecaptchaMinScore)
		return false
	}
	return true
}
