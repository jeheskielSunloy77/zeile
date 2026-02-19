package email

import "fmt"

func (c *Client) SendWelcomeEmail(to, firstName string) error {
	data := map[string]string{
		"UserFirstName": firstName,
	}

	return c.SendEmail(
		to,
		"Welcome to Boilerplate!",
		TemplateWelcome,
		data,
	)
}

func (c *Client) SendEmailVerification(to, username, code string, expiresInMinutes int) error {
	data := map[string]string{
		"Username":         username,
		"VerificationCode": code,
		"ExpiresInMinutes": fmt.Sprintf("%d", expiresInMinutes),
	}

	return c.SendEmail(
		to,
		"Verify your email address",
		TemplateEmailVerification,
		data,
	)
}
