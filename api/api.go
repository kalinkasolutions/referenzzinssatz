package api

import (
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/recaptcha"
	"github.com/kalinkasolutions/referenzzinssatz/repository/interestraterepo"
	"github.com/kalinkasolutions/referenzzinssatz/repository/subscriberrepo"
	"github.com/kalinkasolutions/referenzzinssatz/sendmail"
	"github.com/kalinkasolutions/referenzzinssatz/web"
	"log/slog"
)

const maxEmailLength = 254

var errInvalidEmail = errors.New("invalid email address")

type Api struct {
	subscriberRepo   subscriberrepo.ISubscriberRepository
	interestRateRepo interestraterepo.IInterestRepository
	config           config.Config
	logger           *slog.Logger
	sendMail         sendmail.ISendMail
	captcha          recaptcha.IRecaptcha
}

// page carries the data for every template; each page uses only some of the fields.
type page struct {
	Title            string
	Message          string
	Success          bool
	ContactEmail     string
	ConfirmAction    string
	ConfirmLabel     string
	ReCaptchaSiteKey string
	SourceUrl        string
	CurrentRate      *interestraterepo.InterestRate
}

type subscribeRequest struct {
	Mail           string
	ReCaptchaToken string
}

func NewApi(config config.Config, logger *slog.Logger, subscriberRepo subscriberrepo.ISubscriberRepository, interestRateRepo interestraterepo.IInterestRepository, sendMail sendmail.ISendMail, captcha recaptcha.IRecaptcha) *Api {
	return &Api{
		subscriberRepo:   subscriberRepo,
		interestRateRepo: interestRateRepo,
		config:           config,
		logger:           logger,
		sendMail:         sendMail,
		captcha:          captcha,
	}
}

func (a *Api) Load() {
	if !a.config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router, err := a.Router()
	if err != nil {
		a.logger.Error("Failed to set up router", "error", err)
		os.Exit(1)
	}

	a.logger.Info("Starting API", "port", a.config.Port)
	if err := router.Run(":" + a.config.Port); err != nil {
		a.logger.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func (a *Api) Router() (*gin.Engine, error) {
	templates, err := web.PageTemplates()
	if err != nil {
		return nil, err
	}

	router := gin.New()
	router.Use(requestLogger(a.logger))
	router.Use(gin.Recovery())
	if err := router.SetTrustedProxies(a.config.TrustedProxies); err != nil {
		return nil, err
	}
	router.SetHTMLTemplate(templates)

	router.StaticFS("/static", http.FS(web.Static()))
	router.GET("/", a.home())
	router.POST("/subscribe", a.subscribe())
	router.GET("/validate", a.confirmPage("validationcode", page{
		Title:        "Anmeldung bestätigen",
		Message:      "Möchten Sie per E-Mail informiert werden, sobald sich der Referenzzinssatz ändert?",
		ConfirmLabel: "Ja, Anmeldung bestätigen",
	}))
	router.POST("/validate", a.validate())
	router.GET("/unsubscribe", a.confirmPage("unsubscribecode", page{
		Title:        "Abmelden",
		Message:      "Möchten Sie keine E-Mails mehr zu Änderungen des Referenzzinssatzes erhalten?",
		ConfirmLabel: "Ja, abmelden",
	}))
	router.POST("/unsubscribe", a.unsubscribe())
	router.NoRoute(func(ctx *gin.Context) {
		ctx.HTML(http.StatusNotFound, "message.html", page{
			Title:   "Seite nicht gefunden",
			Message: "Diese Seite gibt es nicht.",
		})
	})
	return router, nil
}

// requestLogger leaves out the query string, which carries the validation and unsubscribe codes.
func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()
		logger.Info("HTTP request",
			"method", ctx.Request.Method,
			"path", ctx.Request.URL.Path,
			"status", ctx.Writer.Status(),
			"durationMs", float64(time.Since(start).Microseconds())/1000,
			"clientIp", ctx.ClientIP(),
		)
	}
}

func (a *Api) home() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		content := page{
			Title:            "Referenzzinssatz",
			ReCaptchaSiteKey: a.config.ReCaptchaSiteKey,
			SourceUrl:        a.config.ReferenzZinssatzUrl,
		}
		if newest, found := a.interestRateRepo.GetNewest(); found {
			content.CurrentRate = &newest
		}
		ctx.HTML(http.StatusOK, "main.html", content)
	}
}

// Every accepted address gets the same answer, so the form does not reveal who is subscribed.
func (a *Api) subscribe() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var request subscribeRequest
		if err := ctx.ShouldBindJSON(&request); err != nil {
			a.logger.Warn("Failed to parse subscribe request", "error", err)
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
			return
		}

		email, err := normalizeEmail(request.Mail)
		if err != nil {
			a.logger.Info("Subscribe request rejected", "reason", "invalid_email")
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid_email"})
			return
		}

		if !a.captcha.CreateAssessment(request.ReCaptchaToken, "subscribe") {
			a.logger.Info("Subscribe request rejected", "reason", "captcha_failed")
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "captcha_failed"})
			return
		}

		// Subscribers are logged by id only, so mail addresses stay out of shipped logs.
		subscriber, found := a.subscriberRepo.GetSubscriberByEmail(email)
		switch {
		case !found:
			subscriber, err = a.subscriberRepo.InsertSubscriber(email)
			if err != nil {
				a.logger.Error("Failed to insert subscriber", "error", err)
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
				return
			}
			a.logger.Info("New subscriber added", "subscriberId", subscriber.Id)
		case subscriber.EmailValidated:
			a.logger.Info("Subscribe request for an already confirmed subscriber", "subscriberId", subscriber.Id)
		default:
			a.logger.Info("Resending validation mail to unconfirmed subscriber", "subscriberId", subscriber.Id)
		}

		// Unconfirmed addresses get the link again, in case the first mail was lost.
		if !subscriber.EmailValidated && !a.sendMail.SendValidationMail(subscriber) {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "mail_failed"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"status": "confirmation_pending"})
	}
}

// Links in mails are opened by scanners and previews too, so a GET only asks
// for confirmation and the change itself needs the POST from this page.
func (a *Api) confirmPage(codeParameter string, content page) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		code := ctx.Query(codeParameter)
		if code == "" {
			a.logger.Info("Link opened without code", "path", ctx.Request.URL.Path)
			ctx.HTML(http.StatusNotFound, "message.html", a.invalidLinkPage())
			return
		}
		confirm := content
		confirm.ConfirmAction = ctx.Request.URL.Path + "?" + url.Values{codeParameter: {code}}.Encode()
		ctx.HTML(http.StatusOK, "confirm.html", confirm)
	}
}

func (a *Api) validate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !a.subscriberRepo.ValidateEmail(ctx.Query("validationcode")) {
			a.logger.Info("Subscription confirmation with unknown code")
			ctx.HTML(http.StatusNotFound, "message.html", page{
				Title:   "Link nicht gültig",
				Message: "Dieser Bestätigungslink ist nicht mehr gültig. Bitte melden Sie sich erneut an.",
			})
			return
		}
		a.logger.Info("Subscription confirmed")
		ctx.HTML(http.StatusOK, "message.html", page{
			Title:   "Anmeldung bestätigt",
			Message: "Sie erhalten ab sofort eine E-Mail, sobald sich der Referenzzinssatz ändert.",
			Success: true,
		})
	}
}

// Also serves one-click unsubscribes (RFC 8058) sent by mail clients.
func (a *Api) unsubscribe() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !a.subscriberRepo.Unsubscribe(ctx.Query("unsubscribecode")) {
			a.logger.Info("Unsubscribe with unknown code")
			ctx.HTML(http.StatusNotFound, "message.html", a.invalidLinkPage())
			return
		}
		a.logger.Info("Subscriber unsubscribed")
		ctx.HTML(http.StatusOK, "message.html", page{
			Title:   "Abgemeldet",
			Message: "Sie erhalten keine E-Mails mehr zu Änderungen des Referenzzinssatzes.",
			Success: true,
		})
	}
}

func (a *Api) invalidLinkPage() page {
	return page{
		Title:        "Link nicht gültig",
		Message:      "Dieser Link ist nicht gültig oder wurde bereits verwendet. Falls Sie weiterhin E-Mails erhalten, schreiben Sie uns bitte mit Ihrer angemeldeten E-Mail-Adresse:",
		ContactEmail: a.config.SMTP_Username,
	}
}

// normalizeEmail accepts a bare address only, no display name or angle brackets.
func normalizeEmail(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) > maxEmailLength {
		return "", errInvalidEmail
	}
	address, err := mail.ParseAddress(trimmed)
	if err != nil || address.Name != "" || address.Address != trimmed {
		return "", errInvalidEmail
	}
	domain := trimmed[strings.LastIndex(trimmed, "@")+1:]
	if !strings.Contains(domain, ".") {
		return "", errInvalidEmail
	}
	return strings.ToLower(trimmed), nil
}
