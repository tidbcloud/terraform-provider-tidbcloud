package tidbcloud

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPreSignedUrlUpload(t *testing.T) {
	for _, tc := range []struct {
		name           string
		insecure       bool
		redirectStatus int
		otherOrigin    bool
		wantErr        bool
	}{
		{name: "HTTPS upload"},
		{name: "HTTP upload", insecure: true, wantErr: true},
		{name: "HTTPS redirect", redirectStatus: http.StatusFound},
		{name: "HTTPS redirect to another origin", redirectStatus: http.StatusFound, otherOrigin: true},
		{name: "HTTP 301 redirect", insecure: true, redirectStatus: http.StatusMovedPermanently, wantErr: true},
		{name: "HTTP 302 redirect", insecure: true, redirectStatus: http.StatusFound, wantErr: true},
		{name: "HTTP 303 redirect", insecure: true, redirectStatus: http.StatusSeeOther, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const content = "id,name\n1,example\n"
			const uploadTarget = "/upload/data%20file%2Bname.csv?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
				"&X-Amz-Credential=example%2F20261105%2Fus-east-1%2Fs3%2Faws4_request" +
				"&X-Amz-Security-Token=example%2Btoken%2Fvalue%3D&X-Amz-Signature=example-signature"
			var plaintextRequests, secureRequests atomic.Int32
			plaintext := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				plaintextRequests.Add(1)
			}))
			defer plaintext.Close()
			other := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				secureRequests.Add(1)
				if r.Header.Get("Authorization") != "" {
					t.Error("upload sent API credentials")
				}
			}))
			defer other.Close()
			secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				secureRequests.Add(1)
				if r.URL.Path != "/redirected" {
					if r.RequestURI != uploadTarget {
						t.Errorf("upload URL changed: got %q, want %q", r.RequestURI, uploadTarget)
					}
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
					}
					if r.Method != http.MethodPut || string(body) != content || r.ContentLength != int64(len(content)) {
						t.Error("upload lost its method, file contents, or content length")
					}
					if tc.redirectStatus != 0 {
						target := "/redirected"
						if tc.insecure {
							target = plaintext.URL
						} else if tc.otherOrigin {
							target = other.URL
						}
						w.Header().Set("Location", target)
						w.WriteHeader(tc.redirectStatus)
					}
				}
			}))
			defer secure.Close()

			previous := http.DefaultTransport
			http.DefaultTransport = secure.Client().Transport
			t.Cleanup(func() { http.DefaultTransport = previous })

			file, err := os.CreateTemp(t.TempDir(), "upload-*.csv")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = file.Close() })
			if _, err := io.WriteString(file, content); err != nil {
				t.Fatal(err)
			}
			if _, err := file.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}

			url := secure.URL + uploadTarget
			if tc.insecure && tc.redirectStatus == 0 {
				url = plaintext.URL
			}
			err = (&ClientDelegate{}).PreSignedUrlUpload(&url, file, int64(len(content)))
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "HTTPS") {
					t.Errorf("expected HTTPS validation error, got %v", err)
				}
			} else if err != nil {
				t.Error(err)
			}
			if plaintextRequests.Load() != 0 {
				t.Error("sent a request over plaintext HTTP")
			}
			var wantRequests int32 = 1
			if tc.insecure && tc.redirectStatus == 0 {
				wantRequests = 0
			} else if !tc.insecure && tc.redirectStatus != 0 {
				wantRequests = 2
			}
			if got := secureRequests.Load(); got != wantRequests {
				t.Errorf("HTTPS requests = %d, want %d", got, wantRequests)
			}
		})
	}
}
