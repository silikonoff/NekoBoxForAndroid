// Package configcompat translates NekoBox's persisted 1.12 configuration to the
// 1.14 schema. It never rewrites the user's stored profile or subscription.
package configcompat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type object = map[string]any

func Normalize(input []byte) ([]byte, error) {
	var config object
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("configuration must be an object")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("trailing configuration data")
	}
	if err := wireguard(config); err != nil {
		return nil, err
	}
	if err := inbounds(config); err != nil {
		return nil, err
	}
	if err := dns(config); err != nil {
		return nil, err
	}
	realityTCPCompatibility(config)
	return json.Marshal(config)
}

// The current Chrome hybrid ClientHello is larger than the legacy 1.12 hello.
// Split its TCP transmission without removing key shares or changing TLS bytes.
// An explicit fragment=false remains an opt-out for advanced configurations.
func realityTCPCompatibility(config object) {
	for _, raw := range list(config["outbounds"]) {
		outbound := obj(raw)
		if str(outbound["type"]) != "vless" || len(obj(outbound["transport"])) != 0 {
			continue
		}
		tls := obj(outbound["tls"])
		if tls["enabled"] != true || obj(tls["reality"])["enabled"] != true {
			continue
		}
		fingerprint := str(obj(tls["utls"])["fingerprint"])
		if fingerprint == "chrome" || fingerprint == "" {
			reality := obj(tls["reality"])
			if _, explicit := reality["legacy_chrome_fallback"]; !explicit {
				reality["legacy_chrome_fallback"] = true
			}
		}
		if _, explicit := tls["fragment"]; !explicit {
			tls["fragment"] = true
			if _, explicit := tls["fragment_fallback_delay"]; !explicit {
				tls["fragment_fallback_delay"] = "50ms"
			}
		}
	}
}

func str(v any) string { s, _ := v.(string); return s }
func list(v any) []any {
	if a, ok := v.([]any); ok {
		return a
	}
	if v == nil {
		return nil
	}
	return []any{v}
}
func obj(v any) object { m, _ := v.(map[string]any); return m }

func inbounds(config object) error {
	var prefix []any
	used := map[string]bool{}
	for _, raw := range list(config["inbounds"]) {
		used[str(obj(raw)["tag"])] = true
	}
	for index, raw := range list(config["inbounds"]) {
		i := obj(raw)
		if i == nil {
			return fmt.Errorf("inbound must be an object")
		}
		strategy := str(i["domain_strategy"])
		sniff, _ := i["sniff"].(bool)
		unmapping, _ := i["udp_disable_domain_unmapping"].(bool)
		if sniff || (strategy != "" && strategy != "as_is") || unmapping {
			tag := str(i["tag"])
			if tag == "" {
				tag = fmt.Sprintf("neko-legacy-inbound-%d", index)
				for used[tag] {
					tag += "_"
				}
				used[tag] = true
				i["tag"] = tag
			}
			if sniff {
				r := object{"inbound": []string{tag}, "action": "sniff"}
				if v := i["sniff_override_destination"]; v != nil {
					r["override_destination"] = v
				}
				if v := i["sniff_timeout"]; v != nil {
					r["timeout"] = v
				}
				prefix = append(prefix, r)
			}
			if strategy != "" && strategy != "as_is" {
				prefix = append(prefix, object{"inbound": []string{tag}, "action": "resolve", "strategy": strategy})
			}
			if unmapping {
				prefix = append(prefix, object{"inbound": []string{tag}, "action": "route-options", "udp_disable_domain_unmapping": true})
			}
		}
		for _, key := range []string{"sniff", "sniff_override_destination", "sniff_timeout", "domain_strategy", "udp_disable_domain_unmapping"} {
			delete(i, key)
		}
		if str(i["type"]) == "tun" {
			for modern, old := range map[string][]string{"address": {"inet4_address", "inet6_address"}, "route_address": {"inet4_route_address", "inet6_route_address"}, "route_exclude_address": {"inet4_route_exclude_address", "inet6_route_exclude_address"}} {
				addresses := list(i[modern])
				for _, key := range old {
					addresses = append(addresses, list(i[key])...)
					delete(i, key)
				}
				if len(addresses) > 0 {
					i[modern] = addresses
				}
			}
			// sing-box 1.12 no longer used this legacy flag. The new default is EIM.
			delete(i, "endpoint_independent_nat")
		}
	}
	if len(prefix) > 0 {
		route := obj(config["route"])
		if route == nil {
			route = object{}
			config["route"] = route
		}
		route["rules"] = append(prefix, list(route["rules"])...)
	}
	return nil
}

func wireguard(config object) error {
	var outbounds []any
	endpoints := list(config["endpoints"])
	for _, value := range list(config["outbounds"]) {
		o := obj(value)
		if str(o["type"]) != "wireguard" {
			outbounds = append(outbounds, value)
			continue
		}
		endpoint := object{}
		for k, v := range o {
			endpoint[k] = v
		}
		if _, conflict := endpoint["address"]; conflict {
			return fmt.Errorf("wireguard %s: address and legacy local_address conflict", str(o["tag"]))
		}
		endpoint["address"] = o["local_address"]
		delete(endpoint, "local_address")
		if o["system_interface"] != nil {
			endpoint["system"] = o["system_interface"]
			delete(endpoint, "system_interface")
		}
		if o["interface_name"] != nil {
			endpoint["name"] = o["interface_name"]
			delete(endpoint, "interface_name")
		}
		var peers []any
		if o["peers"] != nil {
			for _, raw := range list(o["peers"]) {
				peer := obj(raw)
				if peer == nil {
					return fmt.Errorf("wireguard peer must be an object")
				}
				peer["address"] = peer["server"]
				delete(peer, "server")
				peer["port"] = peer["server_port"]
				delete(peer, "server_port")
				if peer["allowed_ips"] == nil {
					peer["allowed_ips"] = []string{"0.0.0.0/0", "::/0"}
				}
				peers = append(peers, peer)
			}
		} else {
			peer := object{"address": o["server"], "port": o["server_port"], "public_key": o["peer_public_key"], "allowed_ips": []string{"0.0.0.0/0", "::/0"}}
			for _, key := range []string{"pre_shared_key", "reserved"} {
				if o[key] != nil {
					peer[key] = o[key]
				}
			}
			peers = []any{peer}
		}
		endpoint["peers"] = peers
		for _, key := range []string{"server", "server_port", "peer_public_key", "pre_shared_key", "reserved"} {
			delete(endpoint, key)
		}
		endpoints = append(endpoints, endpoint)
	}
	if config["outbounds"] != nil {
		config["outbounds"] = outbounds
	}
	if len(endpoints) > 0 {
		config["endpoints"] = endpoints
	}
	return nil
}

func dns(config object) error {
	d := obj(config["dns"])
	if d == nil {
		return nil
	}
	defaultOutbound := str(obj(config["route"])["final"])
	direct := map[string]bool{}
	for _, raw := range list(config["outbounds"]) {
		o := obj(raw)
		if defaultOutbound == "" {
			defaultOutbound = str(o["tag"])
		}
		// The legacy direct detour bypassed the default proxy. In 1.14 an empty
		// direct outbound is deliberately not a valid detour; no detour is direct.
		if str(o["type"]) == "direct" && len(o) == 2 {
			direct[str(o["tag"])] = true
		}
	}
	strategies := map[string]string{}
	subnets := map[string]any{}
	rcodes := map[string]string{}
	var servers []any
	legacy := false
	for _, raw := range list(d["servers"]) {
		s := obj(raw)
		if s == nil {
			return fmt.Errorf("DNS server must be an object")
		}
		if str(s["type"]) != "" && str(s["type"]) != "legacy" {
			servers = append(servers, s)
			continue
		}
		legacy = true
		tag, address := str(s["tag"]), str(s["address"])
		strategies[tag] = str(s["strategy"])
		if s["client_subnet"] != nil {
			subnets[tag] = s["client_subnet"]
		}
		if strings.HasPrefix(address, "rcode://") {
			rcode := strings.TrimPrefix(address, "rcode://")
			rcodes[tag] = map[string]string{"success": "NOERROR", "format_error": "FORMERR", "server_failure": "SERVFAIL", "name_error": "NXDOMAIN", "not_implemented": "NOTIMP", "refused": "REFUSED"}[rcode]
			if rcodes[tag] == "" {
				return fmt.Errorf("unknown legacy DNS rcode %q", rcode)
			}
			continue
		}
		n := object{}
		for k, v := range s {
			n[k] = v
		}
		for _, k := range []string{"address", "strategy", "address_resolver", "address_strategy", "address_fallback_delay", "client_subnet"} {
			delete(n, k)
		}
		// Legacy query policy is migrated below, independently of dialer options.
		resolver := str(s["address_resolver"])
		if resolver != "" {
			n["domain_resolver"] = object{"server": resolver}
			if strategy := str(s["address_strategy"]); strategy != "" {
				obj(n["domain_resolver"])["strategy"] = strategy
			}
		}
		if s["address_fallback_delay"] != nil {
			n["fallback_delay"] = s["address_fallback_delay"]
		}
		detour := str(n["detour"])
		if direct[detour] {
			delete(n, "detour")
		} else if detour == "" && defaultOutbound != "" {
			n["detour"] = defaultOutbound
		}
		switch address {
		case "local":
			n["type"] = "local"
			delete(n, "detour")
			delete(n, "domain_resolver")
		case "fakeip":
			n["type"] = "fakeip"
			delete(n, "detour")
			delete(n, "domain_resolver")
			for _, k := range []string{"inet4_range", "inet6_range"} {
				if v := obj(d["fakeip"])[k]; v != nil {
					n[k] = v
				}
			}
		default:
			u, err := url.Parse(address)
			if err != nil {
				return fmt.Errorf("DNS server %s: %w", tag, err)
			}
			kind, host, port := "udp", address, ""
			if strings.Contains(address, "://") {
				kind = u.Scheme
				host = u.Hostname()
				port = u.Port()
			} else {
				if h, p, err := net.SplitHostPort(address); err == nil {
					host, port = h, p
				} else {
					host = strings.Trim(address, "[]")
				}
			}
			if kind == "h3" {
				kind = "http3"
			}
			switch kind {
			case "udp", "tcp", "tls", "quic", "https", "http3":
			default:
				return fmt.Errorf("DNS server %s: unsupported legacy scheme %q", tag, kind)
			}
			if host == "" {
				return fmt.Errorf("DNS server %s: empty address", tag)
			}
			n["type"], n["server"] = kind, host
			if port != "" {
				p, err := strconv.ParseUint(port, 10, 16)
				if err != nil || p == 0 {
					return fmt.Errorf("DNS server %s: invalid port", tag)
				}
				n["server_port"] = p
			}
			if kind == "https" || kind == "http3" {
				if u.Path != "" && u.Path != "/dns-query" {
					n["path"] = u.RequestURI()
				}
			}
		}
		servers = append(servers, n)
	}
	if !legacy {
		return nil
	}
	d["servers"] = servers
	delete(d, "fakeip")
	var rules []any
	bootstrap := ""
	for _, raw := range list(d["rules"]) {
		r := obj(raw)
		if outbound := list(r["outbound"]); len(outbound) > 0 {
			if len(outbound) != 1 || str(outbound[0]) != "any" || (len(r) != 2 && !(len(r) == 3 && r["strategy"] != nil)) {
				return fmt.Errorf("custom outbound DNS rule needs explicit migration")
			}
			bootstrap = str(r["server"])
			continue
		}
		if err := dnsRule(r, strategies, rcodes, subnets); err != nil {
			return err
		}
		rules = append(rules, r)
	}
	d["rules"] = rules
	if rcodes[str(d["final"])] != "" {
		return fmt.Errorf("rcode DNS server cannot be used as final after migration")
	}
	if strategy := strategies[str(d["final"])]; strategy != "" && d["strategy"] == nil {
		d["strategy"] = strategy
	}
	if subnet := subnets[str(d["final"])]; subnet != nil {
		fallback := object{"server": d["final"], "client_subnet": subnet}
		if strategy := strategies[str(d["final"])]; strategy != "" {
			fallback["strategy"] = strategy
		}
		d["rules"] = append(rules, fallback)
	}
	if bootstrap != "" {
		route := obj(config["route"])
		if route == nil {
			route = object{}
			config["route"] = route
		}
		if route["default_domain_resolver"] == nil {
			resolver := object{"server": bootstrap}
			if strategy := strategies[bootstrap]; strategy != "" {
				resolver["strategy"] = strategy
			}
			if subnet := subnets[bootstrap]; subnet != nil {
				resolver["client_subnet"] = subnet
			}
			route["default_domain_resolver"] = resolver
		}
	}
	return nil
}

func dnsRule(rule object, strategies, rcodes map[string]string, subnets map[string]any) error {
	if rule == nil {
		return fmt.Errorf("DNS rule must be an object")
	}
	if str(rule["type"]) == "logical" {
		for _, raw := range list(rule["rules"]) {
			if err := dnsRule(obj(raw), strategies, rcodes, subnets); err != nil {
				return err
			}
		}
	}
	server := str(rule["server"])
	if subnet := subnets[server]; subnet != nil && rule["client_subnet"] == nil {
		rule["client_subnet"] = subnet
	}
	if rcode := rcodes[server]; rcode != "" {
		rule["action"], rule["rcode"] = "predefined", rcode
		delete(rule, "server")
		// Predefined responses do not make a cached upstream query.
		delete(rule, "disable_cache")
		delete(rule, "strategy")
		delete(rule, "client_subnet")
	} else if strategy := strategies[server]; strategy != "" && rule["strategy"] == nil {
		rule["strategy"] = strategy
	}
	return nil
}
