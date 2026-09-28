package recaptcha

import (
	"context"
	"fmt"
	"os"
	"time"

	recaptcha "cloud.google.com/go/recaptchaenterprise/v2/apiv1"
	recaptchapb "cloud.google.com/go/recaptchaenterprise/v2/apiv1/recaptchaenterprisepb"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/logger"
	"google.golang.org/api/option"
)

type IRecaptcha interface {
	CreateAssessment(token string, recaptchaAction string) bool
}

type Recaptcha struct {
	logger logger.ILogger
	config config.Config
	client *recaptcha.Client
}

func NewReCaptcha(logger logger.ILogger, config config.Config) *Recaptcha {
	client, err := recaptcha.NewClient(context.Background(), option.WithAPIKey(config.RecaptchaGoogleCloudApiKey))
	if err != nil {
		logger.Error("Failed to create reCAPTCHA client: %v", err)
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
		r.logger.Error("Failed to create reCAPTCHA assessment: %v", err)
		return false
	}

	tokenProperties := response.GetTokenProperties()
	if !tokenProperties.GetValid() {
		r.logger.Warning("reCAPTCHA token invalid: %v", tokenProperties.GetInvalidReason())
		return false
	}
	if tokenProperties.GetAction() != recaptchaAction {
		r.logger.Warning("reCAPTCHA action %q does not match expected %q", tokenProperties.GetAction(), recaptchaAction)
		return false
	}

	score := response.GetRiskAnalysis().GetScore()
	if score < r.config.RecaptchaMinScore {
		r.logger.Warning("reCAPTCHA score %.1f below minimum %.1f", score, r.config.RecaptchaMinScore)
		return false
	}
	return true
}
