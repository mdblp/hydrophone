package clients

import (
	"log"
	"net/smtp"
	"strings"
)

type (
	// SmtpNotifier
	SmtpNotifier struct {
		Config *SmtpNotifierConfig
	}

	// SmtpNotifierConfig contains the static configuration for the smtp service
	// Credentials come from the environment and are not passed in via configuration variables.
	SmtpNotifierConfig struct {
		From     string  `json:"fromAddress"`
		ToDomain *string `json:"toDomain,omitempty"`
		Server   string  `json:"serverAdress"`
		Port     string  `json:"serverPort"`
		User     string  `json:"user"`
		Password string  `json:"password"`
	}
)

//NewSmtpNotifier creates a new SMTP notifier (using standard smtp to send emails)
func NewSmtpNotifier(cfg *SmtpNotifierConfig) (*SmtpNotifier, error) {
	if cfg.ToDomain != nil {
		log.Println("SMTP configuration: send mail is restricted to", *cfg.ToDomain)
	}
	return &SmtpNotifier{
		Config: cfg,
	}, nil
}

// Send a message to a list of recipients with a given subject
func (c *SmtpNotifier) Send(to []string, subject string, msg string) (int, string) {
	var toAddress = to[0]
	if c.Config.ToDomain != nil && !strings.HasSuffix(toAddress, *c.Config.ToDomain) {
		log.Println("SMTP e-mail not send to", toAddress, "by server configuration")
		return 200, "OK"
	}
	// Set up authentication information.
	var auth smtp.Auth
	// If no user is provided, then do not try to authenticate to the server (for dev only)
	if c.Config.User != "" {
		auth = smtp.PlainAuth("", c.Config.User, c.Config.Password, c.Config.Server)
	}
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := []byte("To: " + toAddress + "\r\n" +

		"Subject: " + subject + "\r\n" +

		mime + "\r\n" +
		msg + "\r\n")
	err := smtp.SendMail(c.Config.Server+":"+c.Config.Port, auth, c.Config.From, to, body)
	if err != nil {
		log.Println(err.Error())
		return 400, err.Error()
	}
	return 200, "OK"
}
