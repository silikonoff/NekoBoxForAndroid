package configcompat

import (
	"encoding/json"
	"reflect"
	"testing"
)

func normalized(t *testing.T, s string) object {
	t.Helper()
	b, err := Normalize([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	var m object
	if err = json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestLegacyDNSPreservesProxyAndBootstrapPaths(t *testing.T) {
	m := normalized(t, `{"route":{"final":"proxy"},"outbounds":[{"type":"vless","tag":"proxy"},{"type":"direct","tag":"direct"}],"dns":{"final":"dns-remote","servers":[{"tag":"dns-local","address":"local","detour":"direct"},{"tag":"dns-direct","address":"https://resolver.example/dns-query","detour":"direct","address_resolver":"dns-local","strategy":"prefer_ipv4"},{"tag":"dns-remote","address":"tls://1.1.1.1","address_resolver":"dns-direct"}],"rules":[{"outbound":["any"],"server":"dns-direct"},{"domain_suffix":["example.org"],"server":"dns-direct"}]}}`)
	d := obj(m["dns"])
	servers := list(d["servers"])
	if obj(servers[1])["detour"] != nil {
		t.Fatal("empty direct must use direct dialer")
	}
	if obj(servers[2])["detour"] != "proxy" {
		t.Fatal("remote DNS lost proxy path")
	}
	resolver := obj(obj(m["route"])["default_domain_resolver"])
	if resolver["server"] != "dns-direct" || resolver["strategy"] != "prefer_ipv4" {
		t.Fatal("bootstrap resolver lost")
	}
	if len(list(d["rules"])) != 1 || obj(list(d["rules"])[0])["strategy"] != "prefer_ipv4" {
		t.Fatal("DNS rule strategy lost")
	}
}

func TestFakeIPAndBlockedDNS(t *testing.T) {
	m := normalized(t, `{"outbounds":[{"type":"direct","tag":"direct"}],"dns":{"fakeip":{"enabled":true,"inet4_range":"198.18.0.0/15","inet6_range":"fc00::/18"},"servers":[{"tag":"fake","address":"fakeip","strategy":"ipv4_only"},{"tag":"block","address":"rcode://success"}],"rules":[{"domain":["blocked.example"],"server":"block","disable_cache":true},{"inbound":["tun-in"],"server":"fake"}]}}`)
	d := obj(m["dns"])
	servers := list(d["servers"])
	if len(servers) != 1 || obj(servers[0])["inet4_range"] != "198.18.0.0/15" || d["fakeip"] != nil {
		t.Fatal("FakeIP migration failed")
	}
	r := obj(list(d["rules"])[0])
	if r["action"] != "predefined" || r["rcode"] != "NOERROR" || r["server"] != nil {
		t.Fatal("DNS blocking changed")
	}
	if obj(list(d["rules"])[1])["strategy"] != "ipv4_only" {
		t.Fatal("FakeIP IP policy lost")
	}
}

func TestWireGuardKeepsTagChainAndReserved(t *testing.T) {
	m := normalized(t, `{"outbounds":[{"type":"wireguard","tag":"proxy","server":"wg.example","server_port":51820,"local_address":["10.0.0.2/32"],"private_key":"fixture","peer_public_key":"fixture-public","reserved":"AQID","detour":"front","mtu":1280},{"type":"socks","tag":"front","server":"127.0.0.1","server_port":1080}],"route":{"final":"proxy"}}`)
	e := obj(list(m["endpoints"])[0])
	p := obj(list(e["peers"])[0])
	if e["tag"] != "proxy" || e["detour"] != "front" || p["reserved"] != "AQID" || p["address"] != "wg.example" {
		t.Fatal("WireGuard values changed")
	}
	if len(list(m["outbounds"])) != 1 || obj(m["route"])["final"] != "proxy" {
		t.Fatal("WireGuard tag references changed")
	}
}

func TestModernConfigurationIsIdempotent(t *testing.T) {
	input := []byte(`{"outbounds":[{"type":"direct","tag":"direct"}],"dns":{"servers":[{"type":"https","tag":"dns","server":"1.1.1.1"}],"final":"dns"},"experimental":{"custom_integer":18446744073709551615}}`)
	once, err := Normalize(input)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := Normalize(once)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(once, twice) {
		t.Fatal("normalization is not idempotent")
	}
	if !json.Valid(twice) {
		t.Fatal("invalid JSON")
	}
}

func TestMalformedInputIsRejected(t *testing.T) {
	for _, s := range []string{`null`, `{} {}`, `{"dns":{"servers":[{"address":"unhandled://dns.example"}]}}`, `{"dns":{"servers":[{"address":"tls://dns.example:99999"}]}}`} {
		if _, err := Normalize([]byte(s)); err == nil {
			t.Fatalf("accepted %s", s)
		}
	}
}

func TestSniffResolveAndTunAddressesPreserveOrdering(t *testing.T) {
	m := normalized(t, `{"inbounds":[{"type":"tun","tag":"tun-in","inet4_address":["172.19.0.1/30"],"inet6_address":["fdfe::1/126"],"sniff":true,"sniff_override_destination":true,"domain_strategy":"prefer_ipv4","endpoint_independent_nat":true}],"route":{"rules":[{"port":53,"action":"hijack-dns"}]}}`)
	i := obj(list(m["inbounds"])[0])
	if len(list(i["address"])) != 2 || i["inet4_address"] != nil || i["sniff"] != nil {
		t.Fatal("legacy inbound fields were not migrated")
	}
	r := list(obj(m["route"])["rules"])
	if len(r) != 3 || obj(r[0])["action"] != "sniff" || obj(r[0])["override_destination"] != true || obj(r[1])["action"] != "resolve" || obj(r[2])["action"] != "hijack-dns" {
		t.Fatal("legacy rule ordering changed")
	}
}
