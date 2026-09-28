package sendmail

import (
	"bytes"
	"html/template"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kalinkasolutions/referenzzinssatz/config"
	"github.com/kalinkasolutions/referenzzinssatz/repository/interestraterepo"
	"github.com/kalinkasolutions/referenzzinssatz/repository/subscriberrepo"
	"github.com/kalinkasolutions/referenzzinssatz/web"
	"log/slog"
)

const senderName = "Referenzzinssatz"

type ISendMail interface {
	SendValidationMail(subscriber subscriberrepo.Subscriber) bool
	SendReferenzZinssatzUpdate(newest interestraterepo.InterestRate, previous interestraterepo.InterestRate, subscribers []subscriberrepo.Subscriber)
}

type SendMail struct {
	logger    *slog.Logger
	config    config.Config
	templates *template.Template
}

type validationMail struct {
	ValidationUrl string
}

type updateMail struct {
	Newest         interestraterepo.InterestRate
	Previous       interestraterepo.InterestRate
	Change         float64
	SourceUrl      string
	UnsubscribeUrl string
}

type message struct {
	From            mail.Address
	To              string
	Subject         string
	Date            time.Time
	MessageId       string
	ListUnsubscribe string
	HtmlBody        string
}

func NewSendMail(logger *slog.Logger, config config.Config) *SendMail {
	templates, err := web.MailTemplates()
	if err != nil {
		logger.Error("Failed to parse mail templates", "error", err)
		os.Exit(1)
	}
	return &SendMail{
		logger:    logger,
		config:    config,
		templates: templates,
	}
}

func (s *SendMail) SendValidationMail(subscriber subscriberrepo.Subscriber) bool {
	body, err := s.render("validation.html", validationMail{
		ValidationUrl: ValidationUrl(s.config, subscriber),
	})
	if err != nil {
		s.logger.Error("Failed to render validation mail", "error", err)
		return false
	}
	return s.send(s.newMessage(subscriber.Email, "Bitte bestätigen Sie Ihre Anmeldung", body, ""))
}

func (s *SendMail) SendReferenzZinssatzUpdate(newest interestraterepo.InterestRate, previous interestraterepo.InterestRate, subscribers []subscriberrepo.Subscriber) {
	subject := "Der Referenzzinssatz beträgt neu " + web.Percent(newest.ReferenceInterestRate)
	sent := 0
	for _, subscriber := range subscribers {
		unsubscribeUrl := UnsubscribeUrl(s.config, subscriber)
		body, err := s.render("update.html", updateMail{
			Newest:         newest,
			Previous:       previous,
			Change:         newest.ReferenceInterestRate - previous.ReferenceInterestRate,
			SourceUrl:      s.config.ReferenzZinssatzUrl,
			UnsubscribeUrl: unsubscribeUrl,
		})
		if err != nil {
			s.logger.Error("Failed to render update mail", "error", err)
			return
		}
		if s.send(s.newMessage(subscriber.Email, subject, body, unsubscribeUrl)) {
			sent++
		}
	}
	s.logger.Info("Sent reference interest rate update", "sent", sent, "subscribers", len(subscribers))
}

func ValidationUrl(config config.Config, subscriber subscriberrepo.Subscriber) string {
	return config.BaseUrl() + "/validate?validationcode=" + url.QueryEscape(subscriber.ValidationCode)
}

func UnsubscribeUrl(config config.Config, subscriber subscriberrepo.Subscriber) string {
	return config.BaseUrl() + "/unsubscribe?unsubscribecode=" + url.QueryEscape(subscriber.UnsubscribeCode)
}

func (s *SendMail) render(name string, data any) (string, error) {
	var body bytes.Buffer
	err := s.templates.ExecuteTemplate(&body, name, data)
	return body.String(), err
}

func (s *SendMail) newMessage(to string, subject string, htmlBody string, listUnsubscribe string) message {
	return message{
		From:            mail.Address{Name: senderName, Address: s.config.SMTP_Username},
		To:              to,
		Subject:         subject,
		Date:            time.Now(),
		MessageId:       uuid.New().String() + "@" + domainOf(s.config.SMTP_Username),
		ListUnsubscribe: listUnsubscribe,
		HtmlBody:        htmlBody,
	}
}

func (s *SendMail) send(msg message) bool {
	auth := smtp.PlainAuth("", s.config.SMTP_Username, s.config.SMTP_Password, s.config.SMTP_Host)
	err := smtp.SendMail(s.config.SMTP_Host+":"+s.config.SMTP_Port, auth, s.config.SMTP_Username, []string{msg.To}, buildMessage(msg))
	if err != nil {
		s.logger.Error("Failed to send mail", "to", msg.To, "error", err)
		return false
	}
	return true
}

// buildMessage renders an RFC 5322 message; missing Date or Message-ID headers
// and bare LF line endings get mail marked as spam.
func buildMessage(msg message) []byte {
	var out bytes.Buffer
	header := func(name, value string) {
		out.WriteString(name + ": " + value + "\r\n")
	}

	header("From", msg.From.String())
	header("To", (&mail.Address{Address: msg.To}).String())
	header("Subject", mime.QEncoding.Encode("UTF-8", msg.Subject))
	header("Date", msg.Date.Format(time.RFC1123Z))
	header("Message-ID", "<"+msg.MessageId+">")
	if msg.ListUnsubscribe != "" {
		header("List-Unsubscribe", "<"+msg.ListUnsubscribe+">")
		header("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
	}
	header("MIME-Version", "1.0")
	header("Content-Type", `text/html; charset="UTF-8"`)
	header("Content-Transfer-Encoding", "quoted-printable")
	out.WriteString("\r\n")

	body := quotedprintable.NewWriter(&out)
	body.Write([]byte(strings.ReplaceAll(msg.HtmlBody, "\n", "\r\n")))
	body.Close()
	out.WriteString("\r\n")
	return out.Bytes()
}

func domainOf(address string) string {
	if at := strings.LastIndex(address, "@"); at >= 0 {
		return address[at+1:]
	}
	return "localhost"
}
