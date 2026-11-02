package configcompat

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
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
	"github.com/sagernet/sing-box/protocol/direct"
	"github.com/sagernet/sing-box/protocol/vless"
	M "github.com/sagernet/sing/common/metadata"
)

// This is a local sing-box-to-sing-box dataplane test. It does not establish
// compatibility with server-side Xray or an Android VpnService.
func TestLocalVLESSTCPAndUDP(t *testing.T) {
	ctx := context.Background()
	newBox := func(config string) *box.Box {
		t.Helper()
		ins := inbound.NewRegistry()
		vless.RegisterInbound(ins)
		outs := outbound.NewRegistry()
		direct.RegisterOutbound(outs)
		vless.RegisterOutbound(outs)
		dnsRegistry := boxdns.NewTransportRegistry()
		local.RegisterTransport(dnsRegistry)
		bctx := box.Context(ctx, ins, outs, endpoint.NewRegistry(), dnsRegistry, service.NewRegistry(), certificate.NewRegistry())
		var opts option.Options
		if err := opts.UnmarshalJSONContext(bctx, []byte(config)); err != nil {
			t.Fatal(err)
		}
		b, err := box.New(box.Options{Context: bctx, Options: opts})
		if err != nil {
			t.Fatal(err)
		}
		if err = b.Start(); err != nil {
			b.Close()
			t.Fatal(err)
		}
		t.Cleanup(func() { b.Close() })
		return b
	}
	portReservation, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := portReservation.Addr().(*net.TCPAddr).Port
	portReservation.Close()
	const uuid = "00000000-0000-4000-8000-000000000001"
	server := newBox(fmt.Sprintf(`{"log":{"disabled":true},"inbounds":[{"type":"vless","listen":"127.0.0.1","listen_port":%d,"users":[{"uuid":%q}]}],"outbounds":[{"type":"direct","tag":"direct"}]}`, port, uuid))
	client := newBox(fmt.Sprintf(`{"log":{"disabled":true},"outbounds":[{"type":"vless","tag":"proxy","server":"127.0.0.1","server_port":%d,"uuid":%q}],"route":{"final":"proxy"}}`, port, uuid))
	t.Run("TCP", func(t *testing.T) {
		const payload = "neko-core-1.14.4-local-fixture"
		web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, payload) }))
		defer web.Close()
		httpClient := boxapi.CreateProxyHttpClient(client, nil)
		httpClient.Timeout = 5 * time.Second
		defer httpClient.CloseIdleConnections()
		resp, err := httpClient.Get(web.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil || string(body) != payload {
			t.Fatalf("wrong proxied response: %q, %v", body, err)
		}
	})
	t.Run("ResetClosesActiveConnection", func(t *testing.T) {
		echo, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer echo.Close()
		go func() {
			conn, err := echo.Accept()
			if err == nil {
				defer conn.Close()
				io.Copy(conn, conn)
			}
		}()
		conn, err := client.Outbound().Default().DialContext(ctx, "tcp", M.ParseSocksaddr(echo.Addr().String()))
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err = conn.Write([]byte("ready")); err != nil {
			t.Fatal(err)
		}
		buffer := make([]byte, 5)
		if _, err = io.ReadFull(conn, buffer); err != nil || string(buffer) != "ready" {
			t.Fatalf("initial echo: %q, %v", buffer, err)
		}
		// Router.ResetNetwork only flushes caches; Network.ResetNetwork also
		// closes tracked connections, as required by NekoBox's reset action.
		server.Network().ResetNetwork(ctx)
		if _, err = conn.Read(buffer); err == nil {
			t.Fatal("connection survived network reset")
		} else if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
			t.Fatal("reset did not close the active connection before deadline")
		}
	})
	t.Run("UDP", func(t *testing.T) {
		echo, err := net.ListenPacket("udp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer echo.Close()
		go func() {
			b := make([]byte, 512)
			n, addr, err := echo.ReadFrom(b)
			if err == nil {
				echo.WriteTo(b[:n], addr)
			}
		}()
		destination := M.ParseSocksaddr(echo.LocalAddr().String())
		packet, err := client.Outbound().Default().ListenPacket(ctx, destination)
		if err != nil {
			t.Fatal(err)
		}
		defer packet.Close()
		packet.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err = packet.WriteTo([]byte("neko-udp"), echo.LocalAddr()); err != nil {
			t.Fatal(err)
		}
		buffer := make([]byte, 512)
		n, _, err := packet.ReadFrom(buffer)
		if err != nil || string(buffer[:n]) != "neko-udp" {
			t.Fatalf("wrong proxied UDP response: %q, %v", buffer[:n], err)
		}
	})
}
