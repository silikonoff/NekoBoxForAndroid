//go:build with_utls

package configcompat

import (
	"encoding/binary"
	"fmt"
	utls "github.com/metacubex/utls"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// Controlled path: discard one Hello shape, forward allowed bytes unchanged to
// a real Xray. Counts are blocked/forwarded; no credentials or payload logs.
func helloVariantTestProxy(t *testing.T, serverPort int, policy string) (int, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	modern, legacy := new(atomic.Int32), new(atomic.Int32)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				header := make([]byte, 5)
				if _, err := io.ReadFull(conn, header); err != nil {
					return
				}
				size := int(binary.BigEndian.Uint16(header[3:5]))
				if size > 32768 {
					return
				}
				record := append(header, make([]byte, size)...)
				if _, err := io.ReadFull(conn, record[5:]); err != nil {
					return
				}
				spec, err := new(utls.Fingerprinter).FingerprintClientHello(record)
				if err != nil {
					return
				}
				hybrid := false
				for _, extension := range spec.Extensions {
					if shares, ok := extension.(*utls.KeyShareExtension); ok {
						for _, share := range shares.KeyShares {
							if share.Group == utls.X25519MLKEM768 {
								hybrid = true
							}
						}
					}
				}
				conn.SetReadDeadline(time.Time{})
				allowed := !hybrid
				if policy == "compact" {
					allowed = hybrid && len(record) <= 1448
				}
				if policy == "browser" {
					allowed = hybrid && len(record) > 1448
				}
				if !allowed {
					modern.Add(1)
					io.Copy(io.Discard, conn)
					return
				}
				legacy.Add(1)
				server, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", serverPort), time.Second)
				if err != nil {
					return
				}
				defer server.Close()
				if _, err = server.Write(record); err != nil {
					return
				}
				go func() { io.Copy(server, conn); server.Close() }()
				io.Copy(conn, server)
			}()
		}
	}()
	return listener.Addr().(*net.TCPAddr).Port, modern, legacy
}
