package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/mocks"
	"github.com/kalinkasolutions/referenzzinssatz/repository/interestraterepo"
	"github.com/kalinkasolutions/referenzzinssatz/repository/subscriberrepo"
)

type testApi struct {
	router      *gin.Engine
	subscribers *mocks.SubscriberRepositoryMock
	rates       *mocks.InterestRateRepositoryMock
	mail        *mocks.SendMailMock
	captcha     *mocks.RecaptchaMock
}

func newTestApi(t *testing.T, subscribers ...subscriberrepo.Subscriber) testApi {
	gin.SetMode(gin.TestMode)
	test := testApi{
		subscribers: mocks.NewSubscriberRepositoryMock(subscribers...),
		rates:       mocks.NewInterestRateRepositoryMock(),
		mail:        &mocks.SendMailMock{},
		captcha:     &mocks.RecaptchaMock{Human: true},
	}
	api := NewApi(config.Config{SMTP_Username: "service@example.ch"}, mocks.NewLoggerMock(), test.subscribers, test.rates, test.mail, test.captcha)
	router, err := api.Router()
	if err != nil {
		t.Fatalf("failed to build router: %v", err)
	}
	test.router = router
	return test
}

func (test testApi) request(method string, target string, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	test.router.ServeHTTP(recorder, request)
	return recorder
}

func TestNormalizeEmail(t *testing.T) {
	valid := map[string]string{
		"hello@example.ch":     "hello@example.ch",
		"  Hello@Example.CH ":  "hello@example.ch",
		"first.last@sub.ex.ch": "first.last@sub.ex.ch",
	}
	for input, expected := range valid {
		email, err := normalizeEmail(input)
		assert.Equal(t, nil, err)
		assert.Equal(t, expected, email)
	}

	for _, input := range []string{"", "hello", "hello@localhost", "Hello <hello@example.ch>", "<hello@example.ch>", "a@b.ch, c@d.ch", "hello@example.ch\r\nBcc: x@y.ch", strings.Repeat("a", 250) + "@example.ch"} {
		_, err := normalizeEmail(input)
		assert.NotEqual(t, nil, err)
	}
}

func TestSubscribeSendsValidationMail(t *testing.T) {
	test := newTestApi(t)

	response := test.request(http.MethodPost, "/subscribe", `{"mail": " Hello@Example.ch ", "reCaptchaToken": "token"}`)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, 1, len(test.mail.ValidationMails))
	assert.Equal(t, "hello@example.ch", test.mail.ValidationMails[0].Email)
}

func TestSubscribeResendsForUnconfirmedAndIsSilentForConfirmed(t *testing.T) {
	test := newTestApi(t,
		subscriberrepo.Subscriber{Id: "1", Email: "pending@example.ch"},
		subscriberrepo.Subscriber{Id: "2", Email: "confirmed@example.ch", EmailValidated: true},
	)

	pending := test.request(http.MethodPost, "/subscribe", `{"mail": "pending@example.ch", "reCaptchaToken": "token"}`)
	confirmed := test.request(http.MethodPost, "/subscribe", `{"mail": "confirmed@example.ch", "reCaptchaToken": "token"}`)

	assert.Equal(t, http.StatusOK, pending.Code)
	assert.Equal(t, pending.Body.String(), confirmed.Body.String())
	assert.Equal(t, 1, len(test.mail.ValidationMails))
	assert.Equal(t, "1", test.mail.ValidationMails[0].Id)
}

func TestSubscribeRejectsInvalidInput(t *testing.T) {
	test := newTestApi(t)

	invalidEmail := test.request(http.MethodPost, "/subscribe", `{"mail": "not-an-address", "reCaptchaToken": "token"}`)
	malformed := test.request(http.MethodPost, "/subscribe", `{"mail":`)
	test.captcha.Human = false
	bot := test.request(http.MethodPost, "/subscribe", `{"mail": "hello@example.ch", "reCaptchaToken": "token"}`)

	assert.Equal(t, http.StatusBadRequest, invalidEmail.Code)
	assert.Equal(t, true, strings.Contains(invalidEmail.Body.String(), "invalid_email"))
	assert.Equal(t, http.StatusBadRequest, malformed.Code)
	assert.Equal(t, http.StatusBadRequest, bot.Code)
	assert.Equal(t, true, strings.Contains(bot.Body.String(), "captcha_failed"))
	assert.Equal(t, 0, len(test.mail.ValidationMails))
}

func TestSubscribeReportsFailedMail(t *testing.T) {
	test := newTestApi(t)
	test.mail.Fail = true

	response := test.request(http.MethodPost, "/subscribe", `{"mail": "hello@example.ch", "reCaptchaToken": "token"}`)

	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.Equal(t, true, strings.Contains(response.Body.String(), "mail_failed"))
}

func TestValidateNeedsConfirmation(t *testing.T) {
	test := newTestApi(t, subscriberrepo.Subscriber{Id: "1", Email: "hello@example.ch", ValidationCode: "code-1"})

	page := test.request(http.MethodGet, "/validate?validationcode=code-1", "")

	assert.Equal(t, http.StatusOK, page.Code)
	assert.Equal(t, true, strings.Contains(page.Body.String(), `action="/validate?validationcode=code-1"`))
	assert.Equal(t, false, test.subscribers.GetSubscriber("1").EmailValidated)

	confirmed := test.request(http.MethodPost, "/validate?validationcode=code-1", "")

	assert.Equal(t, http.StatusOK, confirmed.Code)
	assert.Equal(t, true, test.subscribers.GetSubscriber("1").EmailValidated)
}

func TestUnsubscribeNeedsConfirmation(t *testing.T) {
	test := newTestApi(t, subscriberrepo.Subscriber{Id: "1", Email: "hello@example.ch", EmailValidated: true, UnsubscribeCode: "code-1"})

	test.request(http.MethodGet, "/unsubscribe?unsubscribecode=code-1", "")
	assert.Equal(t, 1, len(test.subscribers.GetValidatedSubscribers()))

	response := test.request(http.MethodPost, "/unsubscribe?unsubscribecode=code-1", "List-Unsubscribe=One-Click")

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, 0, len(test.subscribers.GetValidatedSubscribers()))
}

func TestInvalidLinksShowContactAddress(t *testing.T) {
	test := newTestApi(t)

	for _, response := range []*httptest.ResponseRecorder{
		test.request(http.MethodGet, "/unsubscribe", ""),
		test.request(http.MethodPost, "/unsubscribe?unsubscribecode=unknown", ""),
	} {
		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Equal(t, true, strings.Contains(response.Body.String(), `href="mailto:service@example.ch"`))
	}
}

func TestHomeShowsCurrentRate(t *testing.T) {
	test := newTestApi(t)
	test.rates.Insert(interestraterepo.InterestRate{ReferenceInterestRate: 1.25, ValidFrom: time.Date(2025, 9, 2, 0, 0, 0, 0, time.UTC)})

	response := test.request(http.MethodGet, "/", "")

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, true, strings.Contains(response.Body.String(), "1,25 %"))
	assert.Equal(t, true, strings.Contains(response.Body.String(), "2. September 2025"))
}

func TestHomeWithoutRatesStillRenders(t *testing.T) {
	test := newTestApi(t)

	response := test.request(http.MethodGet, "/", "")

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, false, strings.Contains(response.Body.String(), `class="rate"`))
}

func TestUnknownPathIsNotFound(t *testing.T) {
	test := newTestApi(t)

	assert.Equal(t, http.StatusNotFound, test.request(http.MethodGet, "/does-not-exist", "").Code)
	assert.Equal(t, http.StatusOK, test.request(http.MethodGet, "/static/main.css", "").Code)
}
