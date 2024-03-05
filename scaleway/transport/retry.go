package transport

import (
	"bytes"
	"context"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/logging"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// DefaultWaitRetryInterval is used to set the retry interval to 0 during acceptance tests
var DefaultWaitRetryInterval *time.Duration

type retryableTransportOptions struct {
	RetryMax     *int
	RetryWaitMax *time.Duration
	RetryWaitMin *time.Duration
}

func NewRetryableTransportWithOptions(defaultTransport http.RoundTripper, options retryableTransportOptions) http.RoundTripper {
	c := retryablehttp.NewClient()
	c.HTTPClient = &http.Client{Transport: defaultTransport}

	// Defaults
	c.RetryMax = 3
	c.RetryWaitMax = 2 * time.Minute
	c.Logger = logging.L
	c.RetryWaitMin = time.Second * 2
	c.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if resp == nil || resp.StatusCode == http.StatusTooManyRequests {
			return true, err
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	// If ErrorHandler is not set, retryablehttp will wrap http errors
	c.ErrorHandler = func(resp *http.Response, err error, _ int) (*http.Response, error) {
		// err is not nil if there was an error while performing request
		// it should be passed, but do not create an error when request contains an error code
		// http errors are handled by sdk coming after this transport
		if err != nil {
			return resp, err
		}
		return resp, nil
	}

	if options.RetryMax != nil {
		c.RetryMax = *options.RetryMax
	}
	if options.RetryWaitMax != nil {
		c.RetryWaitMax = *options.RetryWaitMax
	}
	if options.RetryWaitMin != nil {
		c.RetryWaitMin = *options.RetryWaitMin
	}

	return &retryableTransport{c}
}

// NewRetryableTransport creates a http transport with retry capability.
// TODO Retry logic should be moved in the SDK
func NewRetryableTransport(defaultTransport http.RoundTripper) http.RoundTripper {
	return NewRetryableTransportWithOptions(defaultTransport, retryableTransportOptions{})
}

// client is a bridge between scw.httpClient interface and retryablehttp.Client
type retryableTransport struct {
	*retryablehttp.Client
}

// RoundTrip wraps calling an HTTP method with retries.
func (c *retryableTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var body io.ReadSeeker
	if r.Body != nil {
		bs, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(bs)
	}
	req, err := retryablehttp.NewRequest(r.Method, r.URL.String(), body)
	if err != nil {
		return nil, err
	}
	for key, val := range r.Header {
		req.Header.Set(key, val[0])
	}
	req.GetBody = func() (io.ReadCloser, error) {
		b, err := req.BodyBytes()
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bytes.NewReader(b)), err
	}
	return c.Client.Do(req)
}
