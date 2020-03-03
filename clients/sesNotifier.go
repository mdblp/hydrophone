package clients

import (
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

const (
	// CharSet The character encoding for the email.
	CharSet = "UTF-8"

	// DefaultTextMessage will be sent to non-HTML email clients that receive our messages
	DefaultTextMessage = "You need an HTML client to read this email."
)

type (
	// SesNotifier contains all information needed to send Amazon SES messages
	SesNotifier struct {
		Config *SesNotifierConfig
		SES    *ses.SES
	}

	// SesNotifierConfig contains the static configuration for the Amazon SES service
	// Credentials come from the environment and are not passed in via configuration variables.
	SesNotifierConfig struct {
		From     string  `json:"fromAddress"`
		ToDomain *string `json:"toDomain,omitempty"`
		Region   string  `json:"region"`
		Endpoint string  `json:"serverEndpoint"`
	}
)

//NewSesNotifier creates a new Amazon SES notifier
func NewSesNotifier(cfg *SesNotifierConfig) (*SesNotifier, error) {

	// For SES, if there is a serverEndpoint specified in config, AWS' default is overriden
	myCustomResolver := func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		if service == endpoints.EmailServiceID && cfg.Endpoint != "" {
			return endpoints.ResolvedEndpoint{
				URL:           cfg.Endpoint,
				SigningRegion: "custom-signing-region",
			}, nil
		}

		return endpoints.DefaultResolver().EndpointFor(service, region, optFns...)
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region:           aws.String(cfg.Region),
		EndpointResolver: endpoints.ResolverFunc(myCustomResolver),
	}))

	// Verify whether we have actual credentials (for information tracing)
	// It is looking for credentials in this order:
	// - environment variables AWS_ACCESS_KEY_ID + AWS_ACCESS_SECRET_KEY
	// - existing .aws profile
	// - EC role to be assumed
	// Note: validity of found credentials is not performed at this stage
	creds, err := sess.Config.Credentials.Get()
	if err != nil {
		log.Printf("No AWS credentials were found. Email will not be sent. Error: %s", err.Error())
	} else {
		log.Printf("AWS credentials found with provider %s", creds.ProviderName)
	}

	if cfg.ToDomain != nil {
		log.Println("SES configuration: send mail is restricted to", *cfg.ToDomain)
	}

	return &SesNotifier{
		Config: cfg,
		SES:    ses.New(sess),
	}, nil
}

// Send a message to a list of recipients with a given subject
func (c *SesNotifier) Send(to []string, subject string, msg string) (int, string) {
	var toAwsAddress = make([]*string, len(to))
	for i, x := range to {
		if c.Config.ToDomain != nil && !strings.HasSuffix(x, *c.Config.ToDomain) {
			log.Println("SES e-mail not send to", x, "by server configuration")
			continue
		}
		toAwsAddress[i] = aws.String(x)
	}

	if len(toAwsAddress) == 0 {
		return http.StatusOK, "OK"
	}

	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: toAwsAddress,
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(msg),
				},
				Text: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(DefaultTextMessage),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(CharSet),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(c.Config.From),
	}

	// Attempt to send the email.
	result, err := c.SES.SendEmail(input)

	// Return error messages if they occur. They are traced in the caller function
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			return http.StatusInternalServerError, aerr.Error()
		} else {
			return http.StatusInternalServerError, err.Error()
		}
	}
	return http.StatusOK, result.String()
}
