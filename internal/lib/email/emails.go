// Package email provides utilities for sending templated emails such as
// welcome messages, notifications, and password resets to users.
package email

// SendWelcomeEmail sends a personalized "Welcome" email to a new user.
func (c *Client) SendWelcomeEmail(to, firstName string) error {
	data := map[string]string{
		"UserFirstName": firstName,
	}

	return c.SendEmail(to, "Welcome to TradeAnalyze", TemplateWelcome, data)
}
