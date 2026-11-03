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

func TestLegacyHTTPSRequestPreservesBasePath(t *testing.T) {
	var received atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if r.TLS == nil || r.URL.Path != "/proxy/api/v1beta/projects/1/clusters" || !strings.Contains(string(body), "synthetic-root-password") {
			t.Errorf("unexpected request: TLS=%t path=%s body=%s", r.TLS != nil, r.URL.Path, body)
		}
		if r.UserAgent() != "endpoint-test" {
			t.Errorf("unexpected User-Agent: %s", r.UserAgent())
		}
		received.Store(true)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"1"}`)
	}))
	defer server.Close()
	request := testAPIRequest(t, "legacy", server.URL+"/proxy", server.Client().Transport)
	if err := request(); err != nil {
		t.Fatal(err)
	}
	if !received.Load() {
		t.Fatal("HTTPS request did not reach the custom endpoint")
	}

	client, imports, err := NewApiClient("fake-public", "fake-private", server.URL+"/proxy", "endpoint-test")
	if err != nil {
		t.Fatal(err)
	}
	if client.Transport != imports.Transport || client.Transport.(*httpTransport.Runtime).BasePath != "/proxy" {
		t.Fatal("legacy and import clients must share the configured base path and transport")
	}
}

func TestAPIClientsRejectHTTPSDowngrades(t *testing.T) {
	for _, kind := range apiClientKinds {
		t.Run(kind, func(t *testing.T) {
			for _, status := range []int{http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
				t.Run(fmt.Sprint(status), func(t *testing.T) {
					var plaintextRequests atomic.Int32
					plaintext := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						plaintextRequests.Add(1)
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, `{"id":"1"}`)
					}))
					defer plaintext.Close()
					secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Location", plaintext.URL)
						w.WriteHeader(status)
					}))
					defer secure.Close()
					request := testAPIRequest(t, kind, secure.URL, secure.Client().Transport)
					err := request()
					if err == nil || !strings.Contains(err.Error(), "redirects must use HTTPS") {
						t.Errorf("expected HTTPS redirect error, got %v", err)
					}
					if got := plaintextRequests.Load(); got != 0 {
						t.Fatalf("sent %d requests over plaintext HTTP", got)
					}
				})
			}
		})
	}
}

func TestAPIClientsAllowHTTPSRedirects(t *testing.T) {
	for _, kind := range apiClientKinds {
		t.Run(kind, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if requests.Add(1) == 1 {
					w.Header().Set("Location", "/redirected")
					w.WriteHeader(http.StatusTemporaryRedirect)
					return
				}
				body, _ := io.ReadAll(r.Body)
				if r.Method != http.MethodPost || !strings.Contains(string(body), "synthetic-root-password") {
					t.Errorf("redirect lost the request method or body")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"id":"1"}`)
			}))
			defer server.Close()
			request := testAPIRequest(t, kind, server.URL, server.Client().Transport)
			if err := request(); err != nil {
				t.Fatal(err)
			}
			if got := requests.Load(); got != 2 {
				t.Fatalf("expected an HTTPS redirect, got %d requests", got)
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
	request := testAPIRequest(t, "legacy", server.URL, server.Client().Transport)
	err := request()
	if err == nil || !strings.Contains(err.Error(), "stopped after 10 redirects") {
		t.Fatalf("expected redirect limit, got %v", err)
	}
	if got := requests.Load(); got != 10 {
		t.Fatalf("expected 10 requests, got %d", got)
	}
}

var apiClientKinds = []string{"legacy", "dedicated", "iam", "branch", "cluster", "backup", "import", "export"}

// Exercise each constructor's HTTP client, including the legacy Digest transport.
func testAPIRequest(t *testing.T, kind, endpoint string, transport http.RoundTripper) func() error {
	t.Helper()
	var client *http.Client
	switch kind {
	case "legacy":
		previous := http.DefaultTransport
		http.DefaultTransport = transport
		t.Cleanup(func() { http.DefaultTransport = previous })
		legacy, _, err := NewApiClient("fake-public", "fake-private", endpoint, "endpoint-test")
		if err != nil {
			t.Fatal(err)
		}
		return func() error {
			password := "synthetic-root-password"
			_, err := legacy.Cluster.CreateCluster(cluster.NewCreateClusterParams().WithProjectID("1").WithBody(cluster.CreateClusterBody{
				Config: &cluster.CreateClusterParamsBodyConfig{RootPassword: &password},
			}))
			return err
		}
	case "dedicated":
		dedicated, err := NewDedicatedApiClient(transport, endpoint, "test")
		if err != nil {
			t.Fatal(err)
		}
		client = dedicated.GetConfig().HTTPClient
	case "iam":
		iamClient, err := NewIAMApiClient(transport, endpoint, "test")
		if err != nil {
			t.Fatal(err)
		}
		client = iamClient.GetConfig().HTTPClient
	default:
		branchClient, clusterClient, backupClient, importClient, exportClient, err := NewServerlessApiClient(transport, endpoint, "test")
		if err != nil {
			t.Fatal(err)
		}
		client = map[string]*http.Client{
			"branch": branchClient.GetConfig().HTTPClient, "cluster": clusterClient.GetConfig().HTTPClient,
			"backup": backupClient.GetConfig().HTTPClient, "import": importClient.GetConfig().HTTPClient,
			"export": exportClient.GetConfig().HTTPClient,
		}[kind]
	}
	return func() error {
		resp, err := client.Post(endpoint, "application/json", strings.NewReader(`{"root_password":"synthetic-root-password"}`))
		if resp != nil {
			_ = resp.Body.Close()
		}
		return err
	}
}
