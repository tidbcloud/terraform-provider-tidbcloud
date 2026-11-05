package tidbcloud

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

func validateApiUrl(value string) (*url.URL, error) {
	u, err := url.ParseRequestURI(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		// Do not include the URL or parsing error: either can contain credentials.
		return nil, errors.New("API endpoint must be an absolute HTTPS URL with a hostname")
	}
	return u, nil
}

func newAPIHTTPClient(transport http.RoundTripper) *http.Client {
	client := newHTTPSClient(transport)
	checkHTTPS := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := checkHTTPS(req, via); err != nil {
			return err
		}
		// Digest can authenticate again even if net/http strips Authorization.
		initial := via[0].URL
		if !strings.EqualFold(req.URL.Hostname(), initial.Hostname()) || httpsPort(req.URL) != httpsPort(initial) {
			return errors.New("API redirects must stay on the original host and port")
		}
		return nil
	}
	return client
}

func httpsPort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	return "443"
}

func newHTTPSClient(transport http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "https" {
				return errors.New("API redirects must use HTTPS")
			}
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
	}
}
