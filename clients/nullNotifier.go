package clients

import "log"

type (
	// NullNotifier
	NullNotifier struct {
	}
)

// NewNullNotifier Create a dummy e-mail notifier
func NewNullNotifier() (*NullNotifier, error) {
	log.Println("Mail functionality is disabled, no e-mail will be sent.")
	return &NullNotifier{}, nil
}

// Send do nothing, return 200, "OK"
func (c *NullNotifier) Send(to []string, subject string, msg string) (int, string) {
	var toAddress = to[0]
	log.Printf("Not sending mail to %s, disabled by server configuration\n", toAddress)
	return 200, "OK"
}
