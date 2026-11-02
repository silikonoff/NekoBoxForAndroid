//go:build with_utls

package configcompat

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"net"
	"testing"
	"time"

	boxTLS "github.com/sagernet/sing-box/common/tls"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/logger"
)

// net.Pipe preserves each Write boundary. This checks the actual REALITY
// handshake path, rather than merely inspecting the stored fragment option.
func TestRealityHonorsTCPFragment(t *testing.T) {
	for _, fragment := range []bool{false, true} {
		name := "plain"
		if fragment {
			name = "fragment"
		}
		t.Run(name, func(t *testing.T) {
			config, err := boxTLS.NewRealityClient(context.Background(), logger.NOP(), "127.0.0.1", option.OutboundTLSOptions{
				Enabled: true, ServerName: "example.org", Fragment: fragment,
				UTLS: &option.OutboundUTLSOptions{Enabled: true, Fingerprint: "chrome"},
				Reality: &option.OutboundRealityOptions{Enabled: true,
					PublicKey: base64.RawURLEncoding.EncodeToString([]byte{
						9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
						0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}), ShortID: "0123456789abcdef"},
			})
			if err != nil {
				t.Fatal(err)
			}
			client, server := net.Pipe()
			defer client.Close()
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			done := make(chan struct{})
			go func() { defer close(done); _, _ = boxTLS.ClientHandshake(ctx, client, config) }()
			server.SetReadDeadline(time.Now().Add(time.Second))
			first := make([]byte, 65536)
			n, err := server.Read(first)
			server.Close()
			if err != nil {
				t.Fatal(err)
			}
			if n < 5 || first[0] != 22 {
				t.Fatal("expected TLS handshake record")
			}
			recordSize := 5 + int(binary.BigEndian.Uint16(first[3:5]))
			split := n < recordSize
			t.Logf("first_write=%d record_bytes=%d split=%t", n, recordSize, split)
			if split != fragment {
				t.Fatalf("REALITY ignored fragment=%t", fragment)
			}
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("handshake did not stop")
			}
		})
	}
}
