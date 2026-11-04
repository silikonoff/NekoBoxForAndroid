// Package connectiontest records HTTP test stages without recording endpoints or credentials.
package connectiontest

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"time"
)

type Trace struct {
	start  time.Time
	mu     sync.Mutex
	events []string
}

func New() *Trace { return &Trace{start: time.Now()} }

func (t *Trace) event(stage string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.events) < 64 {
		t.events = append(t.events, fmt.Sprintf("%d:%s", time.Since(t.start).Milliseconds(), stage))
	}
}

type transport struct {
	base  http.RoundTripper
	trace *Trace
}

func (t *transport) CloseIdleConnections() {
	if closer, ok := t.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func (t *Trace) Wrap(base http.RoundTripper) http.RoundTripper {
	return &transport{base: base, trace: t}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := t.trace
	trace := &httptrace.ClientTrace{
		GetConn:  func(string) { r.event("acquire_connection") },
		DNSStart: func(httptrace.DNSStartInfo) { r.event("dns_start") },
		DNSDone: func(i httptrace.DNSDoneInfo) {
			if i.Err != nil {
				r.event("dns_failed")
			} else {
				r.event("dns_done")
			}
		},
		ConnectStart: func(string, string) { r.event("tcp_start") },
		ConnectDone: func(_, _ string, err error) {
			if err != nil {
				r.event("tcp_failed")
			} else {
				r.event("tcp_done")
			}
		},
		GotConn:           func(httptrace.GotConnInfo) { r.event("connection_ready") },
		TLSHandshakeStart: func() { r.event("target_tls_start") },
		TLSHandshakeDone: func(_ tls.ConnectionState, err error) {
			if err != nil {
				r.event("target_tls_failed")
			} else {
				r.event("target_tls_done")
			}
		},
		WroteRequest: func(i httptrace.WroteRequestInfo) {
			if i.Err != nil {
				r.event("request_failed")
			} else {
				r.event("request_sent")
			}
		},
		GotFirstResponseByte: func() { r.event("response_started") },
	}
	// WithClientTrace composes with the timing callbacks installed by speedtest.
	response, err := t.base.RoundTrip(req.WithContext(httptrace.WithClientTrace(req.Context(), trace)))
	if err != nil {
		r.event("http_failed")
	} else {
		r.event("http_done")
	}
	return response, err
}

func (t *Trace) Report(err error) string {
	kind := "none"
	if err != nil {
		kind = "other"
		var dnsErr *net.DNSError
		var netErr net.Error
		message := strings.ToLower(err.Error())
		switch {
		case errors.As(err, &dnsErr):
			kind = "dns"
		case strings.Contains(message, "reality"):
			kind = "reality"
		case strings.Contains(message, "certificate"), strings.Contains(message, "x509"):
			kind = "certificate"
		case strings.Contains(message, "tls"):
			kind = "tls"
		case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
			kind = "timeout"
		case strings.Contains(message, "connection refused"):
			kind = "refused"
		case strings.Contains(message, "network is unreachable"), strings.Contains(message, "no route"):
			kind = "unreachable"
		case errors.Is(err, context.Canceled):
			kind = "cancelled"
		}
	}
	outcome := "pass"
	if err != nil {
		outcome = "fail"
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	// Only fixed tokens and elapsed times are emitted. Never include err.Error(),
	// request URL, DNS answers, addresses, TLS names, profile names or config JSON.
	return fmt.Sprintf("v=1 elapsed_ms=%d outcome=%s kind=%s events=%s", time.Since(t.start).Milliseconds(), outcome, kind, strings.Join(t.events, ","))
}
