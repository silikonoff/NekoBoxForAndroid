package connectiontest

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"
)

func TestTraceSuccessAndExistingCallbacks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	defer server.Close()
	r := New()
	client := &http.Client{Transport: r.Wrap(http.DefaultTransport)}
	called := false
	request, _ := http.NewRequest("GET", server.URL+"/private-token", nil)
	request = request.WithContext(httptrace.WithClientTrace(request.Context(), &httptrace.ClientTrace{GotFirstResponseByte: func() { called = true }}))
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if !called {
		t.Fatal("existing timing callback lost")
	}
	report := r.Report(nil)
	for _, expected := range []string{"outcome=pass", "connection_ready", "request_sent", "response_started"} {
		if !strings.Contains(report, expected) {
			t.Fatalf("missing %s: %s", expected, report)
		}
	}
	if strings.Contains(report, server.URL) || strings.Contains(report, "private-token") {
		t.Fatal("endpoint leaked")
	}
}

func TestTraceTimeoutIdentifiesWaitAfterRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	r := New()
	client := &http.Client{Transport: r.Wrap(http.DefaultTransport), Timeout: 150 * time.Millisecond}
	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("expected timeout")
	}
	report := r.Report(err)
	if !strings.Contains(report, "kind=timeout") || !strings.Contains(report, "request_sent") || strings.Contains(report, "response_started") {
		t.Fatal(report)
	}
}

func TestTraceDNSFailureDoesNotExposeErrorContent(t *testing.T) {
	r := New()
	base := &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) {
		return nil, &net.DNSError{Err: "private-uuid vless://private-password@private-server", Name: "private-sni", Server: "private-dns"}
	}}
	client := &http.Client{Transport: r.Wrap(base)}
	_, err := client.Get("http://private-host/private-token")
	report := r.Report(err)
	if !strings.Contains(report, "kind=dns") || !strings.Contains(report, "acquire_connection") || strings.Contains(report, "private") {
		t.Fatal(report)
	}
	if strings.Contains(New().Report(errors.New("reality: private-pbk")), "private") {
		t.Fatal("error leaked")
	}
}

func TestTraceCertificateFailure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	defer server.Close()
	r := New()
	client := &http.Client{Transport: r.Wrap(&http.Transport{})}
	_, err := client.Get(server.URL)
	report := r.Report(err)
	if err == nil || !strings.Contains(report, "target_tls_failed") || !strings.Contains(report, "kind=certificate") {
		t.Fatal(report)
	}
}
