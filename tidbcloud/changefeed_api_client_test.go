package tidbcloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestChangefeedDelegate(srv *httptest.Server) *DedicatedClientDelegate {
	return &DedicatedClientDelegate{
		httpClient:        srv.Client(),
		changefeedBaseURL: srv.URL + DedicatedAPIBasePath,
	}
}

func TestChangefeedRequestsHitExpectedEndpoints(t *testing.T) {
	type recorded struct {
		method string
		path   string
		query  string
	}
	var last recorded
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last = recorded{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cf-1","clusterId":"c-1","name":"n","replicationCapacity":"4rcu","downstreamType":"KAFKA","state":"RUNNING"}`))
	}))
	defer srv.Close()
	d := newTestChangefeedDelegate(srv)
	ctx := context.Background()

	cf, err := d.GetChangefeed(ctx, "cf-1")
	if err != nil {
		t.Fatalf("GetChangefeed failed: %v", err)
	}
	if cf.Id == nil || *cf.Id != "cf-1" || cf.State == nil || *cf.State != ChangefeedStateRunning {
		t.Errorf("GetChangefeed returned unexpected changefeed: %+v", cf)
	}
	if last.method != http.MethodGet || last.path != "/v1beta1/changefeeds/cf-1" {
		t.Errorf("GetChangefeed hit %s %s", last.method, last.path)
	}

	pageSize := int32(10)
	downstreamType := "KAFKA"
	if _, err := d.ListChangefeeds(ctx, &ListChangefeedsParams{ClusterId: "c-1", DownstreamType: &downstreamType, PageSize: &pageSize}); err != nil {
		t.Fatalf("ListChangefeeds failed: %v", err)
	}
	if last.path != "/v1beta1/changefeeds" || last.query != "clusterId=c-1&downstreamType=KAFKA&pageSize=10" {
		t.Errorf("ListChangefeeds hit %s?%s", last.path, last.query)
	}

	if _, err := d.CreateChangefeed(ctx, &CreateChangefeedRequest{Changefeed: &Changefeed{ClusterId: "c-1", Name: "n"}}); err != nil {
		t.Fatalf("CreateChangefeed failed: %v", err)
	}
	if last.method != http.MethodPost || last.path != "/v1beta1/changefeeds" {
		t.Errorf("CreateChangefeed hit %s %s", last.method, last.path)
	}

	for name, call := range map[string]func() error{
		"pause":  func() error { return d.PauseChangefeed(ctx, "cf-1") },
		"resume": func() error { return d.ResumeChangefeed(ctx, "cf-1") },
		"scale": func() error {
			_, err := d.ScaleChangefeed(ctx, "cf-1", "8rcu")
			return err
		},
		"editDownstreamConfig": func() error {
			_, err := d.EditChangefeedDownstreamConfig(ctx, "cf-1", &EditChangefeedDownstreamConfigRequest{DownstreamType: ChangefeedDownstreamTypeKafka})
			return err
		},
	} {
		if err := call(); err != nil {
			t.Fatalf("%s failed: %v", name, err)
		}
		want := "/v1beta1/changefeeds/cf-1:" + name
		if last.method != http.MethodPost || last.path != want {
			t.Errorf("%s hit %s %s, want POST %s", name, last.method, last.path, want)
		}
	}

	if err := d.DeleteChangefeed(ctx, "cf-1"); err != nil {
		t.Fatalf("DeleteChangefeed failed: %v", err)
	}
	if last.method != http.MethodDelete || last.path != "/v1beta1/changefeeds/cf-1" {
		t.Errorf("DeleteChangefeed hit %s %s", last.method, last.path)
	}
}

func TestChangefeedScaleBodySerialization(t *testing.T) {
	var body map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode scale body: %v", err)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	d := newTestChangefeedDelegate(srv)

	if _, err := d.ScaleChangefeed(context.Background(), "cf-1", "8rcu"); err != nil {
		t.Fatalf("ScaleChangefeed failed: %v", err)
	}
	if body["replicationCapacity"] != "8rcu" {
		t.Errorf("scale body = %v, want replicationCapacity=8rcu", body)
	}
}

func TestIsChangefeedNotFoundError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"changefeed not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()
	d := newTestChangefeedDelegate(srv)

	_, err := d.GetChangefeed(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected an error for 404 response")
	}
	if !IsChangefeedNotFoundError(err) {
		t.Errorf("expected 404 to be reported as not found, got %v", err)
	}
	if IsChangefeedNotFoundError(&ChangefeedAPIError{StatusCode: http.StatusInternalServerError}) {
		t.Error("expected 500 ChangefeedAPIError not to be reported as not found")
	}
	if IsChangefeedNotFoundError(errors.New("some other error")) {
		t.Error("expected a plain error not to be reported as not found")
	}
}
