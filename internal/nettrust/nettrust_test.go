package nettrust

import (
	"context"
	"errors"
	"net/netip"
	"slices"
	"testing"
)

type fakeProber struct {
	via   map[string]string // host name -> answering LAN address
	reach map[string]bool
}

func (f fakeProber) DirectVia(_ context.Context, p Peer) (netip.Addr, bool) {
	raw, ok := f.via[p.HostName]
	if !ok {
		return netip.Addr{}, false
	}
	return netip.MustParseAddr(raw), true
}

func (f fakeProber) Reach(_ context.Context, target string) error {
	if f.reach[target] {
		return nil
	}
	return errors.New("refused")
}

var policy = Policy{
	HomeSubnets:       []string{"192.168.8.0/24"},
	TrustedWifis:      []string{"bakasifu-5Ghz", "bakasifu-2.4Ghz", "fizzlipuzzli"},
	ExitNode:          "home-router",
	SiteRouterTrust:   true,
	SiteRouterTargets: []string{"192.168.8.1:53"},
}

func wifi(ssid, keyMgmt, subnet string) *Network {
	return &Network{Device: "wlan0", SSID: ssid, KeyMgmt: keyMgmt, Subnets: []netip.Prefix{netip.MustParsePrefix(subnet)}}
}

func router(name, route string) Peer {
	return Peer{HostName: name, Online: true, IPs: []netip.Addr{netip.MustParseAddr("100.64.0.9")}, Routes: []netip.Prefix{netip.MustParsePrefix(route)}}
}

func decide(t *testing.T, p Policy, n *Network, tn Tailnet, pr Prober) Verdict {
	t.Helper()
	v, err := Decide(context.Background(), p, n, tn, pr)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestTrustedHomeWifiKeepsHomeLocal(t *testing.T) {
	v := decide(t, policy, wifi("bakasifu-5Ghz", "wpa-psk", "192.168.8.0/24"), Tailnet{Running: true}, fakeProber{})
	if v.Trust != TrustHome || v.ExitNode != "" || !slices.Equal(v.Bypass, []string{"192.168.8.0/24"}) {
		t.Fatalf("verdict = %+v", v)
	}
}

func TestClonedOpenHomeWifiIsUntrusted(t *testing.T) {
	v := decide(t, policy, wifi("bakasifu-5Ghz", "", "192.168.8.0/24"), Tailnet{Running: true}, fakeProber{})
	if v.Trust != TrustUntrusted || v.ExitNode != "home-router" || len(v.Bypass) != 0 {
		t.Fatalf("verdict = %+v", v)
	}
	if len(v.Warnings) != 2 {
		t.Fatalf("want open-wifi and subnet-collision warnings, got %q", v.Warnings)
	}
}

func TestHomeRouterProofWorksOnEthernet(t *testing.T) {
	n := &Network{Device: "enp1s0", Subnets: []netip.Prefix{netip.MustParsePrefix("192.168.8.0/24")}}
	tn := Tailnet{Running: true, Peers: []Peer{router("home-router", "192.168.8.0/24")}}
	v := decide(t, policy, n, tn, fakeProber{via: map[string]string{"home-router": "192.168.8.1"}})
	if v.Trust != TrustHome || v.Router != "home-router via 192.168.8.1" {
		t.Fatalf("verdict = %+v", v)
	}
}

func TestRelayedRouterIsNoProof(t *testing.T) {
	// A foreign LAN reusing 192.168.8.0/24 while the real router answers
	// only from elsewhere.
	n := &Network{Device: "enp1s0", Subnets: []netip.Prefix{netip.MustParsePrefix("192.168.8.0/24")}}
	tn := Tailnet{Running: true, Peers: []Peer{router("home-router", "192.168.8.0/24")}}
	v := decide(t, policy, n, tn, fakeProber{via: map[string]string{"home-router": "203.0.113.7"}})
	if v.Trust != TrustUntrusted || len(v.Bypass) != 0 {
		t.Fatalf("verdict = %+v", v)
	}
}

func TestSiteRouterNeedsEveryTarget(t *testing.T) {
	n := &Network{Device: "enp1s0", Subnets: []netip.Prefix{netip.MustParsePrefix("10.20.0.0/24")}}
	tn := Tailnet{Running: true, Peers: []Peer{router("mum-router", "10.20.0.0/24")}}
	pr := fakeProber{via: map[string]string{"mum-router": "10.20.0.1"}}

	v := decide(t, policy, n, tn, pr)
	if v.Trust != TrustUntrusted || v.ExitNode != "home-router" {
		t.Fatalf("router without home access must not be trusted: %+v", v)
	}

	pr.reach = map[string]bool{"192.168.8.1:53": true}
	v = decide(t, policy, n, tn, pr)
	if v.Trust != TrustSite || v.ExitNode != "" || !slices.Equal(v.Bypass, []string{"10.20.0.0/24"}) {
		t.Fatalf("verdict = %+v", v)
	}
}

func TestExitNodeRouteIsNoSiteProof(t *testing.T) {
	n := &Network{Device: "enp1s0", Subnets: []netip.Prefix{netip.MustParsePrefix("10.20.0.0/24")}}
	tn := Tailnet{Running: true, Peers: []Peer{router("exit", "0.0.0.0/0")}}
	pr := fakeProber{via: map[string]string{"exit": "10.20.0.5"}, reach: map[string]bool{"192.168.8.1:53": true}}
	if v := decide(t, policy, n, tn, pr); v.Trust != TrustUntrusted {
		t.Fatalf("verdict = %+v", v)
	}
}

func TestSiteRouterTrustOffFallsBackToWifi(t *testing.T) {
	p := policy
	p.SiteRouterTrust = false
	tn := Tailnet{Running: true, Peers: []Peer{router("sis-router", "192.168.1.0/24")}}
	pr := fakeProber{via: map[string]string{"sis-router": "192.168.1.1"}, reach: map[string]bool{"192.168.8.1:53": true}}
	v := decide(t, p, wifi("fizzlipuzzli", "sae", "192.168.1.0/24"), tn, pr)
	if v.Trust != TrustTrustedWifi || v.ExitNode != "" {
		t.Fatalf("verdict = %+v", v)
	}
}

func TestUnmanagedPolicyKeepsLegacyBypass(t *testing.T) {
	v := decide(t, Policy{HomeSubnets: []string{"192.168.8.0/24"}}, nil, Tailnet{}, fakeProber{})
	if v.Trust != TrustLegacy || !slices.Equal(v.Bypass, []string{"192.168.8.0/24"}) || v.ManageExitNode {
		t.Fatalf("verdict = %+v", v)
	}
}

func TestStoppedTailscaleLeavesExitNodeAlone(t *testing.T) {
	v := decide(t, policy, wifi("cafe", "", "172.16.0.0/16"), Tailnet{}, fakeProber{})
	if v.Trust != TrustUntrusted || v.ManageExitNode {
		t.Fatalf("verdict = %+v", v)
	}
}

func TestParseDefaultRoutePrefersLowestMetric(t *testing.T) {
	dev, gw, ok, err := parseDefaultRoute([]byte(`[
		{"dst":"default","gateway":"10.0.0.1","dev":"wlan0","metric":600},
		{"dst":"default","gateway":"192.168.8.1","dev":"enp1s0","metric":100}]`))
	if err != nil || !ok || dev != "enp1s0" || gw.String() != "192.168.8.1" {
		t.Fatalf("got %s %s %v %v", dev, gw, ok, err)
	}
}

func TestParseSubnetsMasksGlobalAddresses(t *testing.T) {
	got, err := parseSubnets([]byte(`[{"addr_info":[
		{"family":"inet","local":"192.168.8.23","prefixlen":24,"scope":"global"},
		{"family":"inet6","local":"fe80::1","prefixlen":64,"scope":"link"}]}]`))
	if err != nil || len(got) != 1 || got[0].String() != "192.168.8.0/24" {
		t.Fatalf("got %v %v", got, err)
	}
}

func TestParsePongVia(t *testing.T) {
	out := []byte("pong from r (100.64.0.9) via DERP(fra) in 30ms\npong from r (100.64.0.9) via 192.168.8.1:41641 in 2ms\n")
	if via, ok := parsePongVia(out); !ok || via.String() != "192.168.8.1" {
		t.Fatalf("got %s %v", via, ok)
	}
	if _, ok := parsePongVia([]byte("pong from r (100.64.0.9) via DERP(fra) in 30ms\n")); ok {
		t.Fatal("DERP path counted as direct")
	}
}

func TestParseTailnet(t *testing.T) {
	tn, err := parseTailnet([]byte(`{"BackendState":"Running","Peer":{"k":{
		"HostName":"home-router","DNSName":"home-router.tail1234.ts.net.","TailscaleIPs":["100.64.0.9"],
		"Online":true,"PrimaryRoutes":["192.168.8.0/24"],"ExitNode":true}}}`))
	if err != nil || !tn.Running || len(tn.Peers) != 1 {
		t.Fatalf("got %+v %v", tn, err)
	}
	cur := tn.CurrentExitNode()
	if cur == nil || !cur.Matches("home-router") || !cur.Matches("100.64.0.9") || !cur.Matches("home-router.tail1234.ts.net") {
		t.Fatalf("exit node match failed: %+v", cur)
	}
}

func TestApplyExitNodeOnlyChangesOnDifference(t *testing.T) {
	var calls [][]string
	s := System{run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		return nil, nil
	}}
	tn, _ := parseTailnet([]byte(`{"BackendState":"Running","Peer":{"k":{"HostName":"home-router","ExitNode":true}}}`))
	if changed, _ := s.ApplyExitNode(context.Background(), tn, "home-router"); changed || len(calls) != 0 {
		t.Fatalf("unchanged exit node was set again: %v", calls)
	}
	if changed, _ := s.ApplyExitNode(context.Background(), tn, ""); !changed || !slices.Equal(calls[0], []string{"tailscale", "set", "--exit-node="}) {
		t.Fatalf("calls = %v", calls)
	}
}
