package clients

import "log"

// DummyNotifier empty struct
type DummyNotifier struct {
}

// NewDummyNotifier Create a dummy e-mail notifier
func NewDummyNotifier() (*DummyNotifier, error) {
	log.Println("Mail functionality is disabled, no e-mail will be sent.")
	return &DummyNotifier{}, nil
}

// Send do nothing, return 200, "OK"
func (c *DummyNotifier) Send(to []string, subject string, msg string) (int, string) {
	var toAddress = to[0]
	log.Printf("Not sending mail to %s, disabled by server configuration\n", toAddress)
	return 200, "OK"
}
