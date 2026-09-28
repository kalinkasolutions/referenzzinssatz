package sendmail

import (
	"io"
	"log/slog"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/repository/interestraterepo"
	"github.com/kalinkasolutions/referenzzinssatz/repository/subscriberrepo"
)

var testConfig = config.Config{
	Domain:              "zins.example.ch",
	Ssl:                 true,
	SMTP_Username:       "service@example.ch",
	ReferenzZinssatzUrl: config.DefaultReferenzZinssatzUrl,
}

func TestBuildMessageIsWellFormed(t *testing.T) {
	raw := buildMessage(message{
		From:            mail.Address{Name: senderName, Address: "service@example.ch"},
		To:              "hello@example.ch",
		Subject:         "Bitte bestätigen Sie Ihre Anmeldung",
		Date:            time.Date(2025, 9, 2, 8, 0, 0, 0, time.UTC),
		MessageId:       "id@example.ch",
		ListUnsubscribe: "https://zins.example.ch/unsubscribe?unsubscribecode=abc",
		HtmlBody:        "<p>Grüsse</p>\n<p>Änderung</p>",
	})

	headerEnd := strings.Index(string(raw), "\r\n\r\n")
	assert.NotEqual(t, -1, headerEnd)
	assert.Equal(t, false, strings.Contains(strings.ReplaceAll(string(raw[:headerEnd]), "\r\n", ""), "\n"))

	parsed, err := mail.ReadMessage(strings.NewReader(string(raw)))
	assert.Equal(t, nil, err)

	subject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
	assert.Equal(t, nil, err)
	assert.Equal(t, "Bitte bestätigen Sie Ihre Anmeldung", subject)
	assert.Equal(t, `"Referenzzinssatz" <service@example.ch>`, parsed.Header.Get("From"))
	assert.Equal(t, "<hello@example.ch>", parsed.Header.Get("To"))
	assert.Equal(t, "<id@example.ch>", parsed.Header.Get("Message-ID"))
	assert.Equal(t, "Tue, 02 Sep 2025 08:00:00 +0000", parsed.Header.Get("Date"))
	assert.Equal(t, "<https://zins.example.ch/unsubscribe?unsubscribecode=abc>", parsed.Header.Get("List-Unsubscribe"))
	assert.Equal(t, "List-Unsubscribe=One-Click", parsed.Header.Get("List-Unsubscribe-Post"))

	body, _ := io.ReadAll(quotedprintable.NewReader(parsed.Body))
	assert.Equal(t, "<p>Grüsse</p>\r\n<p>Änderung</p>\r\n", string(body))
}

func TestBuildMessageOmitsUnsubscribeHeadersWhenNotGiven(t *testing.T) {
	raw := string(buildMessage(message{To: "hello@example.ch", Date: time.Now()}))

	assert.Equal(t, false, strings.Contains(raw, "List-Unsubscribe"))
}

func TestMailLinks(t *testing.T) {
	subscriber := subscriberrepo.Subscriber{ValidationCode: "v-1", UnsubscribeCode: "u-1"}

	assert.Equal(t, "https://zins.example.ch/validate?validationcode=v-1", ValidationUrl(testConfig, subscriber))
	assert.Equal(t, "https://zins.example.ch/unsubscribe?unsubscribecode=u-1", UnsubscribeUrl(testConfig, subscriber))
}

func TestRenderUpdateMail(t *testing.T) {
	sendMail := NewSendMail(slog.New(slog.DiscardHandler), testConfig)
	newest := interestraterepo.InterestRate{
		ReferenceInterestRate:     1.25,
		ValidFrom:                 time.Date(2025, 9, 2, 0, 0, 0, 0, time.UTC),
		UnderlyingAvgInterestRate: 1.37,
		ReferenceDateOfSurvey:     time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
	}

	body, err := sendMail.render("update.html", updateMail{
		Newest:         newest,
		Previous:       interestraterepo.InterestRate{ReferenceInterestRate: 1.5},
		Change:         -0.25,
		SourceUrl:      testConfig.ReferenzZinssatzUrl,
		UnsubscribeUrl: "https://zins.example.ch/unsubscribe?unsubscribecode=u-1",
	})

	assert.Equal(t, nil, err)
	for _, expected := range []string{"1,25 %", "2. September 2025", "bisher 1,5 %", "−0,25 Prozentpunkte", "1,37 %", "30.06.2025", "unsubscribecode=u-1"} {
		if !strings.Contains(body, expected) {
			t.Errorf("update mail is missing %q", expected)
		}
	}
}
