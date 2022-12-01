package hydrophone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/mdblp/go-common/clients/status"
	appContext "github.com/mdblp/go-common/context"
	"github.com/mdblp/go-common/errors"
	"github.com/mdblp/hydrophone/models"
)

type (
	ClientInterface interface {
		CancelSignup(confirm models.Confirmation, authToken string) error
		SendNotification(ctx context.Context, topic string, notif interface{}, authToken string) error
		GetNotifications(ctx context.Context, userID string, authToken string) ([]models.Confirmation, error)
	}

	Client struct {
		host       string       // host url
		httpClient *http.Client // store a reference to the http client so we can reuse it
	}

	ClientBuilder struct {
		host       string       // host url
		httpClient *http.Client // store a reference to the http client so we can reuse it
	}
)

func NewHydrophoneClientBuilder() *ClientBuilder {
	return &ClientBuilder{}
}

// WithHost set the host
func (b *ClientBuilder) WithHost(host string) *ClientBuilder {
	b.host = host
	return b
}

// WithHTTPClient set the HTTP client
func (b *ClientBuilder) WithHTTPClient(httpClient *http.Client) *ClientBuilder {
	b.httpClient = httpClient
	return b
}

// Build return client from builder
func (b *ClientBuilder) Build() *Client {

	if b.host == "" {
		panic("Hydrophone client requires a host to be set")
	}
	if b.httpClient == nil {
		b.httpClient = http.DefaultClient
	}

	return &Client{
		httpClient: b.httpClient,
		host:       b.host,
	}
}

// NewHydrophoneClientFromEnv read the config from the environment variables
func NewHydrophoneClientFromEnv(httpClient *http.Client) *Client {
	builder := NewHydrophoneClientBuilder()
	host, _ := os.LookupEnv("HYDROPHONE_HOST")
	return builder.WithHost(host).
		WithHTTPClient(httpClient).
		Build()
}

func (client *Client) getHost() (*url.URL, error) {
	if client.host == "" {
		return nil, errors.New("No client host defined")
	}
	theURL, err := url.Parse(client.host)
	if err != nil {
		return nil, fmt.Errorf("unable to parse urlString[%s]", client.host)
	}
	return theURL, nil
}

func (client *Client) getFullRequestWithContext(ctx context.Context, method string, authToken string, payload interface{}, queryParams map[string]string, pathParams ...string) (*http.Request, error) {
	host, err := client.getHost()
	if err != nil {
		return nil, err
	}
	pathFragments := append([]string{host.Path}, pathParams...)
	host.Path = path.Join(pathFragments...)
	q := host.Query()
	for key, value := range queryParams {
		if value != "" {
			q.Set(key, value)
		}
	}
	host.RawQuery = q.Encode()
	var req *http.Request
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		req, err = http.NewRequestWithContext(ctx, method, host.String(), bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
	} else {
		req, err = http.NewRequestWithContext(ctx, method, host.String(), nil)
		if err != nil {
			return nil, err
		}
	}
	/*eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9 is the default shoreline token header*/
	client.setAuthHeader(req, authToken)
	if traceSessionId, ok := appContext.GetTraceSessionIdCtx(ctx); ok {
		req.Header.Set("x-tidepool-trace-session", traceSessionId)
	}

	return req, nil
}

// Set the correct auth header based on the client configuration
func (client *Client) setAuthHeader(req *http.Request, authToken string) {
	/*eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9 is the default shoreline token header*/
	if strings.Index(authToken, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9") == 0 {
		req.Header.Add("x-tidepool-session-token", authToken)
	} else {
		req.Header.Add("Authorization", "Bearer "+authToken)
	}
}

// Handle various error codes
func handleErrors(res *http.Response, req *http.Request) error {
	if res.StatusCode == 401 {
		return fmt.Errorf("access to Hydrophone is not authorized")
	} else {
		return fmt.Errorf("unknown response code from service[%s], code %v", req.URL, res.StatusCode)
	}
}

func (h *Client) SendNotification(ctx context.Context, topic string, notif interface{}, authToken string) error {
	//logger := appContext.GetLogger(ctx)
	data, err := json.Marshal(notif)
	if err != nil {
		return errors.Wrap(err, "Failure to marshal notification")
	}
	req, err := h.getFullRequestWithContext(ctx, "POST", authToken, data, map[string]string{}, "notifications", topic)
	if err != nil {
		return errors.Wrap(err, "SendNotification: error formatting request")
	}

	res, err := h.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Failure to send notification to hydrophone")
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return &status.StatusError{
			Status: status.NewStatusf(res.StatusCode, "Unknown response code from service[%s]", req.URL),
		}
	}
}

func (client *Client) GetNotifications(ctx context.Context, userID string, authToken string) ([]models.Confirmation, error) {
	logger := appContext.GetLogger(ctx)
	req, err := client.getFullRequestWithContext(ctx, "GET", authToken, nil, map[string]string{}, "notifications", userID)
	if err != nil {
		return nil, errors.Wrap(err, "GetNotifications: error formatting request")
	}

	res, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		var retVal []models.Confirmation
		if err := json.NewDecoder(res.Body).Decode(&retVal); err != nil {
			logger.Error(err)
			return nil, fmt.Errorf("error parsing JSON results: %v", err)
		}
		return retVal, nil
	}
	return nil, handleErrors(res, req)
}
