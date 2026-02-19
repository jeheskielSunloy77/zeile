package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"

	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"gopkg.in/gomail.v2"
)

type Client struct {
	dialer    *gomail.Dialer
	fromEmail string
	fromName  string
	logger    *zerolog.Logger
}

func NewClient(cfg *config.Config, logger *zerolog.Logger) *Client {
	dialer := gomail.NewDialer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password)
	dialer.TLSConfig = &tls.Config{
		ServerName:         cfg.SMTP.Host,
		InsecureSkipVerify: false,
	}

	return &Client{
		dialer:    dialer,
		fromEmail: cfg.SMTP.FromEmail,
		fromName:  cfg.SMTP.FromName,
		logger:    logger,
	}
}

func (c *Client) SendEmail(to, subject string, templateName Template, data map[string]string) error {
	tmplPath := fmt.Sprintf("%s/%s.html", "templates/emails", templateName)

	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return errors.Wrapf(err, "failed to parse email template %s", templateName)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return errors.Wrapf(err, "failed to execute email template %s", templateName)
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", fmt.Sprintf("%s <%s>", c.fromName, c.fromEmail))
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body.String())

	if err := c.dialer.DialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
