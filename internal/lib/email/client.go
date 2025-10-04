// Package email provides functionality for sending HTML-based emails using the Resend API.
// It integrates with Go's standard HTML templating to render dynamic email bodies, and then
// delivers them through Resend's email delivery service. The package is designed to be
// reusable across the application by abstracting away the email client initialization,
// template rendering, and request construction.
package email

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/Barry-dE/go-backend-boilerplate/internal/config"
	"github.com/pkg/errors"
	"github.com/resend/resend-go/v2"
	"github.com/rs/zerolog"
)

type Client struct {
	client *resend.Client
	logger *zerolog.Logger
}

// NewClient initializes and returns a new email Client.
func NewClient(cfg *config.Config, logger *zerolog.Logger) *Client {
	return &Client{
		client: resend.NewClient(cfg.Integration.ResendAPIKey),
		logger: logger,
	}
}

// SendEmail renders an HTML template with dynamic data and sends it via the Resend API.
// Parameters:
// - to: recipient email address.
// - subject: subject line for the email.
// - templateName: name of the email template file (without path).
// - data: key-value pairs passed into the HTML template for rendering.
func (c *Client) SendEmail(to, subject string, templateName Template, data map[string]string) error {

	// Build full path to the HTML template file (e.g., "templates/emails/welcome.html").
	templatePath := fmt.Sprintf("%s/%s.html", "templates/emails", templateName)

	// Parse the template file from the given path.
	templ, err := template.ParseFiles(templatePath)
	if err != nil {
		return errors.Wrapf(err, "failed to parse email template %s", templateName)
	}
	// Execute the parsed template with the provided data and write the result into a buffer.
	var body bytes.Buffer
	if err := templ.Execute(&body, data); err != nil {
		return errors.Wrapf(err, "failed to execute email template %s", templateName)
	}

	//  Build the Resend SendEmailRequest object with the rendered HTML body and other parameters.
	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", "Go-Boilerplate", "onboarding@resend.dev"),
		To:      []string{to},
		Subject: subject,
		Html:    body.String(),
	}

	// Send the email using the Resend client.
	_, err = c.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
