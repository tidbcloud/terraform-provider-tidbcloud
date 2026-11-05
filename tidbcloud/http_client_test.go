package tidbcloud

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/c4pt0r/go-tidbcloud-sdk-v1/client/cluster"
	httpTransport "github.com/go-openapi/runtime/client"
	"github.com/icholy/digest"
)

func TestAPIEndpointValidation(t *testing.T) {
	constructors := map[string]func(string) error{
		"legacy": func(endpoint string) error {
			_, _, err := NewApiClient("fake-public", "fake-private", endpoint, "test")
			return err
		},
		"dedicated": func(endpoint string) error {
			_, err := NewDedicatedApiClient(nil, endpoint, "test")
			return err
		},
		"serverless": func(endpoint string) error {
			_, _, _, _, _, err := NewServerlessApiClient(nil, endpoint, "test")
			return err
		},
		"iam": func(endpoint string) error {
			_, err := NewIAMApiClient(nil, endpoint, "test")
			return err
		},
	}
	cases := []struct {
		name     string
		endpoint string
		valid    bool
	}{
		{"default", DefaultApiUrl, true},
		{"custom host and port", "https://custom.example:8443", true},
		{"base path", "https://custom.example:8443/proxy", true},
		{"http", "http://example.com", false},
		{"relative", "/api", false},
		{"scheme relative", "//example.com/api", false},
		{"opaque", "https:example.com", false},
		{"missing host", "https:///api", false},
		{"port only", "https://:443", false},
		{"parse error with credentials", "https://user:credential-marker@example.com/%zz", false},
	}
	for name, constructor := range constructors {
		t.Run(name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					err := constructor(tc.endpoint)
					if (err == nil) != tc.valid {
						t.Fatalf("valid = %t, error = %v", tc.valid, err)
					}
					if err != nil && strings.Contains(err.Error(), "credential-marker") {
						t.Fatal("endpoint validation disclosed URL credentials")
					}
				})
			}
		})
	}
}

func TestDefaultAPIEndpoints(t *testing.T) {
	dc, err := NewDedicatedApiClient(nil, "", "test")
	if err != nil || dc.GetConfig().Host != "dedicated.tidbapi.com" {
		t.Fatalf("unexpected dedicated endpoint: %v", err)
	}
	bc, sc, brc, ic, ec, err := NewServerlessApiClient(nil, "", "test")
	if err != nil {
		t.Fatal(err)
	}
	for _, host := range []string{bc.GetConfig().Host, sc.GetConfig().Host, brc.GetConfig().Host, ic.GetConfig().Host, ec.GetConfig().Host} {
		if host != "serverless.tidbapi.com" {
			t.Fatalf("unexpected serverless endpoint: %s", host)
		}
	}
	iamClient, err := NewIAMApiClient(nil, "", "test")
	if err != nil || iamClient.GetConfig().Host != "iam.tidbapi.com" {
		t.Fatalf("unexpected IAM endpoint: %v", err)
	}
}

func TestLegacyAPIHTTPSRequests(t *testing.T) {
	for _, tc := range []struct {
		name           string
		redirectStatus int
		insecure       bool
	}{
		{"direct HTTPS", 0, false},
		{"HTTPS redirect", http.StatusTemporaryRedirect, false},
		{"HTTP 302", http.StatusFound, true},
		{"HTTP 307", http.StatusTemporaryRedirect, true},
		{"HTTP 308", http.StatusPermanentRedirect, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var plaintextRequests, secureRequests atomic.Int32
			plaintext := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				plaintextRequests.Add(1)
			}))
			defer plaintext.Close()
			secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				secureRequests.Add(1)
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if r.TLS == nil || r.Method != http.MethodPost || !strings.Contains(string(body), "synthetic-root-password") || r.UserAgent() != "endpoint-test" {
					t.Error("HTTPS request lost its method, body, or User-Agent")
				}
				switch r.URL.Path {
				case "/proxy/api/v1beta/projects/1/clusters":
					if tc.redirectStatus != 0 {
						target := "/redirected"
						if tc.insecure {
							target = plaintext.URL
						}
						w.Header().Set("Location", target)
						w.WriteHeader(tc.redirectStatus)
						return
					}
				case "/redirected":
				default:
					t.Errorf("unexpected request path: %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"id":"1"}`)
			}))
			defer secure.Close()

			previous := http.DefaultTransport
			http.DefaultTransport = secure.Client().Transport
			t.Cleanup(func() { http.DefaultTransport = previous })
			client, imports, err := NewApiClient("fake-public", "fake-private", secure.URL+"/proxy", "endpoint-test")
			if err != nil {
				t.Fatal(err)
			}
			if client.Transport != imports.Transport || client.Transport.(*httpTransport.Runtime).BasePath != "/proxy" {
				t.Fatal("legacy and import clients must share the configured transport and base path")
			}
			password := "synthetic-root-password"
			_, err = client.Cluster.CreateCluster(cluster.NewCreateClusterParams().WithProjectID("1").WithBody(cluster.CreateClusterBody{
				Config: &cluster.CreateClusterParamsBodyConfig{RootPassword: &password},
			}))
			if tc.insecure {
				if err == nil || !strings.Contains(err.Error(), "redirects must use HTTPS") {
					t.Fatalf("expected HTTPS redirect error, got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if plaintextRequests.Load() != 0 {
				t.Fatal("sent a request over plaintext HTTP")
			}
			var wantRequests int32 = 1
			if tc.redirectStatus != 0 && !tc.insecure {
				wantRequests = 2
			}
			if got := secureRequests.Load(); got != wantRequests {
				t.Fatalf("HTTPS requests = %d, want %d", got, wantRequests)
			}
		})
	}
}

func TestAPIRedirectLimit(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Location", "/loop")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	client := newAPIHTTPClient(server.Client().Transport)
	resp, err := client.Get(server.URL)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "stopped after 10 redirects") {
		t.Fatalf("expected redirect limit, got %v", err)
	}
	if got := requests.Load(); got != 10 {
		t.Fatalf("expected 10 requests, got %d", got)
	}
}

func TestAPIAuthenticationRedirects(t *testing.T) {
	for _, tc := range []struct {
		name     string
		endpoint string
		target   string
		allowed  bool
	}{
		{"relative path", "https://custom.example", "/redirected", true},
		{"custom port", "https://custom.example:8443", "https://custom.example:8443/redirected", true},
		{"explicit default port", "https://custom.example", "https://custom.example:443/redirected", true},
		{"implicit default port", "https://custom.example:443", "https://custom.example/redirected", true},
		{"hostname case", "https://custom.example", "https://CUSTOM.EXAMPLE/redirected", true},
		{"IPv6 default port", "https://[::1]", "https://[::1]:443/redirected", true},
		{"different host", "https://custom.example", "https://other.example/redirected", false},
		{"subdomain", "https://custom.example", "https://sub.custom.example/redirected", false},
		{"different port", "https://custom.example:8443", "https://custom.example:9443/redirected", false},
		{"default to custom port", "https://custom.example", "https://custom.example:8443/redirected", false},
	} {
		for _, status := range []int{301, 302, 303, 307, 308} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, status), func(t *testing.T) {
				var initialAuthenticated bool
				var redirectedRequests, authenticatedRedirects int
				transport := &digest.Transport{
					Username: "fake-public",
					Password: "fake-private",
					Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
						resp := &http.Response{
							StatusCode: http.StatusOK,
							Header:     make(http.Header),
							Body:       http.NoBody,
							Request:    r,
						}
						if r.URL.Path == "/redirected" {
							redirectedRequests++
						}
						if r.Header.Get("Authorization") == "" {
							resp.StatusCode = http.StatusUnauthorized
							resp.Header.Set("WWW-Authenticate", `Digest realm="test", nonce="test-nonce", algorithm=MD5, qop="auth"`)
							return resp, nil
						}
						if r.URL.Path == "/start" {
							initialAuthenticated = true
							resp.StatusCode = status
							resp.Header.Set("Location", tc.target)
						} else {
							authenticatedRedirects++
						}
						return resp, nil
					}),
				}
				client := newAPIHTTPClient(NewTransportWithAgent(transport, "test"))
				resp, err := client.Get(tc.endpoint + "/start")
				if resp != nil {
					_ = resp.Body.Close()
				}
				if !initialAuthenticated {
					t.Fatal("failed to authenticate to the configured endpoint")
				}
				if tc.allowed {
					if err != nil || authenticatedRedirects != 1 {
						t.Fatalf("expected authenticated redirect, got %d requests and error %v", authenticatedRedirects, err)
					}
				} else {
					if err == nil || !strings.Contains(err.Error(), "original host and port") {
						t.Errorf("expected cross-origin redirect error, got %v", err)
					}
					if redirectedRequests != 0 {
						t.Errorf("sent %d requests outside the configured origin", redirectedRequests)
					}
				}
			})
		}
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
