package configcompat

import (
	"context"
	"testing"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter/certificate"
	"github.com/sagernet/sing-box/adapter/endpoint"
	"github.com/sagernet/sing-box/adapter/inbound"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/adapter/service"
	boxdns "github.com/sagernet/sing-box/dns"
	"github.com/sagernet/sing-box/dns/transport"
	"github.com/sagernet/sing-box/dns/transport/fakeip"
	"github.com/sagernet/sing-box/dns/transport/local"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/protocol/direct"
	"github.com/sagernet/sing-box/protocol/mixed"
	"github.com/sagernet/sing-box/protocol/tun"
	"github.com/sagernet/sing-box/protocol/vless"
	wireguardProtocol "github.com/sagernet/sing-box/protocol/wireguard"
)

// These are synthetic legacy fixtures, not exports from a tested Android APK.
// Decode with the real pinned core so a syntactically valid but unusable schema
// migration cannot pass merely by comparing JSON maps.
func TestMigratedProfilesDecodeWithCore114(t *testing.T) {
	ins := inbound.NewRegistry()
	tun.RegisterInbound(ins)
	mixed.RegisterInbound(ins)
	outs := outbound.NewRegistry()
	direct.RegisterOutbound(outs)
	vless.RegisterOutbound(outs)
	eps := endpoint.NewRegistry()
	wireguardProtocol.RegisterEndpoint(eps)
	dnsReg := boxdns.NewTransportRegistry()
	transport.RegisterUDP(dnsReg)
	transport.RegisterTCP(dnsReg)
	transport.RegisterTLS(dnsReg)
	transport.RegisterHTTPS(dnsReg)
	local.RegisterTransport(dnsReg)
	fakeip.RegisterTransport(dnsReg)
	ctx := box.Context(context.Background(), ins, outs, eps, dnsReg, service.NewRegistry(), certificate.NewRegistry())
	fixtures := []string{
		`{"inbounds":[{"type":"tun","tag":"tun-in","inet4_address":["172.19.0.1/30"],"sniff":true,"sniff_override_destination":true,"domain_strategy":"prefer_ipv4","stack":"gvisor"}],"outbounds":[{"type":"vless","tag":"proxy","server":"test.invalid","server_port":443,"uuid":"00000000-0000-4000-8000-000000000001","flow":"xtls-rprx-vision","tls":{"enabled":true,"server_name":"test.invalid","utls":{"enabled":true,"fingerprint":"chrome"},"reality":{"enabled":true,"public_key":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","short_id":"0123"}}},{"type":"direct","tag":"direct"}],"route":{"final":"proxy","rules":[{"port":53,"action":"hijack-dns"}]},"dns":{"final":"remote","independent_cache":true,"servers":[{"address":"local","tag":"local","detour":"direct"},{"address":"1.1.1.1","tag":"direct-dns","detour":"direct","address_resolver":"local"},{"address":"https://dns.example/dns-query","tag":"remote","address_resolver":"direct-dns","strategy":"prefer_ipv4"},{"address":"rcode://success","tag":"block"}],"rules":[{"outbound":["any"],"server":"direct-dns"},{"domain":["blocked.invalid"],"server":"block","disable_cache":true}]}}`,
		`{"outbounds":[{"type":"wireguard","tag":"proxy","server":"wg.invalid","server_port":51820,"local_address":["10.0.0.2/32"],"private_key":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","peer_public_key":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","reserved":"AQID","mtu":1280}],"route":{"final":"proxy"},"dns":{"final":"fake","fakeip":{"enabled":true,"inet4_range":"198.18.0.0/15","inet6_range":"fc00::/18"},"servers":[{"address":"fakeip","tag":"fake","strategy":"ipv4_only"}],"rules":[{"server":"fake","inbound":["tun-in"],"disable_cache":true}]}}`,
	}
	for i, input := range fixtures {
		normalized, err := Normalize([]byte(input))
		if err != nil {
			t.Fatalf("fixture %d normalization: %v", i, err)
		}
		var opts option.Options
		if err := opts.UnmarshalJSONContext(ctx, normalized); err != nil {
			t.Fatalf("fixture %d core schema: %v", i, err)
		}
	}
}
