package configcompat

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter/certificate"
	"github.com/sagernet/sing-box/adapter/endpoint"
	"github.com/sagernet/sing-box/adapter/inbound"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/adapter/service"
	"github.com/sagernet/sing-box/boxapi"
	boxdns "github.com/sagernet/sing-box/dns"
	"github.com/sagernet/sing-box/dns/transport/local"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/protocol/vless"
	M "github.com/sagernet/sing/common/metadata"
)

// Optional isolated interoperability test. No production server/profile is used.
// Keys are generated in memory and the server config is supplied through stdin.
func TestXrayRealityVisionHTTPS(t *testing.T) {
	binary := os.Getenv("NEKO_XRAY_BINARY")
	if binary == "" {
		t.Skip("set NEKO_XRAY_BINARY to the checksum-verified Xray server executable")
	}
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	privateKey := base64.RawURLEncoding.EncodeToString(key.Bytes())
	publicKey := base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
	fallback := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	fallback.EnableHTTP2 = true
	fallback.Config.ErrorLog = log.New(io.Discard, "", 0)
	fallback.TLS = &tls.Config{MinVersion: tls.VersionTLS13}
	fallback.StartTLS()
	defer fallback.Close()
	reservation, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := reservation.Addr().(*net.TCPAddr).Port
	reservation.Close()
	const uuid = "00000000-0000-4000-8000-000000000001"
	config := object{"log": object{"loglevel": "error"}, "inbounds": []any{object{"listen": "127.0.0.1", "port": port, "protocol": "vless", "settings": object{"decryption": "none", "clients": []any{object{"id": uuid, "flow": "xtls-rprx-vision"}}}, "streamSettings": object{"network": "raw", "security": "reality", "realitySettings": object{"show": false, "target": fallback.Listener.Addr().String(), "serverNames": []string{"example.org"}, "privateKey": privateKey, "shortIds": []string{"0123456789abcdef"}}}}}, "outbounds": []any{object{"protocol": "freedom", "tag": "direct"}}}
	obj(config["log"])["loglevel"] = "debug"
	// The isolated fixture's HTTPS echo target is on loopback. Current Xray
	// blocks private destinations by default; permit only this loopback range.
	obj(list(config["outbounds"])[0])["settings"] = object{"finalRules": []any{object{"action": "allow", "network": "tcp", "ip": []string{"127.0.0.1/32"}}}}
	serverJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "run", "-config", "stdin:")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	cmd.Stdin = bytes.NewReader(serverJSON)
	var serverLogs bytes.Buffer
	cmd.Stdout = &serverLogs
	cmd.Stderr = &serverLogs
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	t.Cleanup(func() {
		cmd.Process.Kill()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
		if t.Failed() {
			safe := strings.ReplaceAll(serverLogs.String(), privateKey, "[redacted]")
			if len(safe) > 3500 {
				safe = safe[len(safe)-3500:]
			}
			t.Logf("Xray server diagnostics: %s", safe)
		}
	})
	ready := false
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		select {
		case err := <-done:
			t.Fatalf("Xray test server exited: %v", err)
		default:
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ready {
		t.Fatal("Xray test server did not become ready")
	}
	outs := outbound.NewRegistry()
	vless.RegisterOutbound(outs)
	dnsRegistry := boxdns.NewTransportRegistry()
	local.RegisterTransport(dnsRegistry)
	ctx := box.Context(context.Background(), inbound.NewRegistry(), outs, endpoint.NewRegistry(), dnsRegistry, service.NewRegistry(), certificate.NewRegistry())
	clientJSON := fmt.Sprintf(`{"log":{"disabled":true},"outbounds":[{"type":"vless","tag":"proxy","server":"127.0.0.1","server_port":%d,"uuid":%q,"flow":"xtls-rprx-vision","tls":{"enabled":true,"server_name":"example.org","utls":{"enabled":true,"fingerprint":"chrome"},"reality":{"enabled":true,"public_key":%q,"short_id":"0123456789abcdef"}}}],"route":{"final":"proxy"}}`, port, uuid, publicKey)
	normalized, err := Normalize([]byte(clientJSON))
	if err != nil {
		t.Fatal(err)
	}
	var opts option.Options
	if err = opts.UnmarshalJSONContext(ctx, normalized); err != nil {
		t.Fatal(err)
	}
	client, err := box.New(box.Options{Context: ctx, Options: opts})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err = client.Start(); err != nil {
		t.Fatal(err)
	}
	probe, err := client.Outbound().Default().DialContext(context.Background(), "tcp", M.ParseSocksaddr(fallback.Listener.Addr().String()))
	if err != nil {
		t.Fatalf("outer REALITY authentication: %v", err)
	}
	probe.Close()
	t.Log("outer REALITY authentication succeeded")
	echo := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read before writing: HTTP/1 otherwise drains the unread request body
		// when committing response headers, truncating an io.Copy echo fixture.
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "fixture read failed", 500)
			return
		}
		w.Write(body)
	}))
	defer echo.Close()
	httpClient := boxapi.CreateProxyHttpClient(client, nil)
	httpClient.Timeout = 20 * time.Second
	httpClient.Transport.(*http.Transport).TLSHandshakeTimeout = 15 * time.Second
	defer httpClient.CloseIdleConnections()
	pool := x509.NewCertPool()
	pool.AddCert(echo.Certificate())
	httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	payload := bytes.Repeat([]byte("neko-reality-vision-fixture"), 4096)
	control, err := echo.Client().Post(echo.URL, "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	controlBody, err := io.ReadAll(control.Body)
	control.Body.Close()
	if err != nil || !bytes.Equal(controlBody, payload) {
		t.Fatal("direct HTTPS echo control failed")
	}
	resp, err := httpClient.Post(echo.URL, "application/octet-stream", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(len(payload)+1)))
	if err != nil || !bytes.Equal(body, payload) {
		t.Fatalf("HTTPS payload mismatch: got %d bytes, error %v", len(body), err)
	}
	t.Logf("HTTPS upload/download matched %d bytes", len(payload))
}
