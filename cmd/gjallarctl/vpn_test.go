package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeVPNPaths(t *testing.T) (reapplied *int) {
	t.Helper()
	dir := t.TempDir()
	oldStatus, oldPolicy, oldOverride, oldReapply, oldEUID := vpnStatusPath, vpnPolicyPath, vpnOverridePath, vpnReapply, vpnEUID
	t.Cleanup(func() {
		vpnStatusPath, vpnPolicyPath, vpnOverridePath, vpnReapply, vpnEUID = oldStatus, oldPolicy, oldOverride, oldReapply, oldEUID
	})
	vpnStatusPath = filepath.Join(dir, "status.json")
	vpnPolicyPath = filepath.Join(dir, "policy.json")
	vpnOverridePath = filepath.Join(dir, "state", "override.json")
	vpnEUID = func() int { return 0 }
	n := 0
	vpnReapply = func() error { n++; return nil }
	write := func(path, data string) {
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(vpnPolicyPath, `{"homeSubnets":["192.168.8.0/24"],"trustedWifis":["home"],"exitNode":"OpenWrt","siteRouterTrust":false,"siteRouterTargets":[]}`)
	write(vpnStatusPath, `{"trust":"untrusted","reason":"r","bypass":[],"exitNode":"OpenWrt","manageExitNode":true}`)
	return &n
}

func TestVPNOverrideCommands(t *testing.T) {
	reapplied := fakeVPNPaths(t)
	var out, errOut bytes.Buffer
	for _, args := range [][]string{
		{"trust-wifi", "Shi 2,4"},
		{"untrust-wifi", "home"},
		{"exit-node", "auto"},
	} {
		if code := runVPN(args, &out, &errOut); code != 0 {
			t.Fatalf("%q: code %d: %s", args, code, errOut.String())
		}
	}
	if *reapplied != 3 {
		t.Fatalf("reapplied %d times", *reapplied)
	}
	if !strings.Contains(out.String(), `override:  trust "Shi 2,4", untrust "home", exit node auto`) {
		t.Fatalf("status:\n%s", out.String())
	}

	out.Reset()
	if code := runVPN([]string{"export"}, &out, &errOut); code != 0 {
		t.Fatalf("export: %s", errOut.String())
	}
	want := `tailscale = { enable = true; homeSubnets = [ "192.168.8.0/24" ]; trustedWifis = [ "Shi 2,4" ]; exitNode = "auto"; siteRouterTrust = false; siteRouterTargets = [ ]; };`
	if strings.TrimSpace(out.String()) != want {
		t.Fatalf("export = %s", out.String())
	}

	for _, args := range [][]string{{"trust-wifi", "home"}, {"untrust-wifi", "Shi 2,4"}, {"exit-node", "default"}} {
		if code := runVPN(args, &out, &errOut); code != 0 {
			t.Fatalf("%q: %s", args, errOut.String())
		}
	}
	if _, err := os.Stat(vpnOverridePath); !os.IsNotExist(err) {
		t.Fatalf("override left behind after undoing every change: %v", err)
	}
}

func TestVPNOverrideNeedsRoot(t *testing.T) {
	reapplied := fakeVPNPaths(t)
	vpnEUID = func() int { return 1000 }
	var out, errOut bytes.Buffer
	if code := runVPN([]string{"exit-node", "off"}, &out, &errOut); code != 1 || *reapplied != 0 {
		t.Fatalf("code %d, reapplied %d", code, *reapplied)
	}
	if _, err := os.Stat(vpnOverridePath); !os.IsNotExist(err) {
		t.Fatal("non-root wrote an override")
	}
}
