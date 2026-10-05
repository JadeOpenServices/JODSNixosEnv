package nettrust

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Override holds network trust changes made at runtime with gjallarctl.
// It is layered over the policy built into the system, so a rebuild keeps
// it, and `gjallarctl vpn export` turns it into generated state.
type Override struct {
	// TrustWifis are added to the policy's trusted Wi-Fi names.
	TrustWifis []string `json:"trustWifis,omitempty"`
	// UntrustWifis are removed from the policy's trusted Wi-Fi names.
	UntrustWifis []string `json:"untrustWifis,omitempty"`
	// ExitNode replaces the policy's exit node when set.
	ExitNode *string `json:"exitNode,omitempty"`
}

func (o Override) Empty() bool {
	return len(o.TrustWifis) == 0 && len(o.UntrustWifis) == 0 && o.ExitNode == nil
}

// LoadOverride reads an override file; a missing file is no override.
func LoadOverride(path string) (Override, error) {
	var o Override
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return o, nil
	}
	if err != nil {
		return o, fmt.Errorf("read network trust override: %w", err)
	}
	if err := json.Unmarshal(data, &o); err != nil {
		return o, fmt.Errorf("parse network trust override %s: %w", path, err)
	}
	return o, nil
}

// SaveOverride replaces the override file atomically. An empty override
// removes it.
func SaveOverride(path string, o Override) error {
	if o.Empty() {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Apply layers the override over p.
func (o Override) Apply(p Policy) Policy {
	trusted := []string{}
	for _, ssid := range append(slices.Clone(p.TrustedWifis), o.TrustWifis...) {
		if !slices.Contains(o.UntrustWifis, ssid) && !slices.Contains(trusted, ssid) {
			trusted = append(trusted, ssid)
		}
	}
	p.TrustedWifis = trusted
	if o.ExitNode != nil {
		p.ExitNode = *o.ExitNode
	}
	return p
}

// Trust marks ssid trusted, base being the policy built into the system.
func (o *Override) Trust(base Policy, ssid string) {
	o.UntrustWifis = without(o.UntrustWifis, ssid)
	o.TrustWifis = without(o.TrustWifis, ssid)
	if !slices.Contains(base.TrustedWifis, ssid) {
		o.TrustWifis = append(o.TrustWifis, ssid)
	}
}

// Untrust stops trusting ssid, base being the policy built into the system.
func (o *Override) Untrust(base Policy, ssid string) {
	o.TrustWifis = without(o.TrustWifis, ssid)
	o.UntrustWifis = without(o.UntrustWifis, ssid)
	if slices.Contains(base.TrustedWifis, ssid) {
		o.UntrustWifis = append(o.UntrustWifis, ssid)
	}
}

// SetExitNode overrides the exit node; "default" returns to the policy's.
func (o *Override) SetExitNode(base Policy, name string) error {
	name = strings.TrimSpace(name)
	switch {
	case name == "" || strings.ContainsAny(name, " \t\n"):
		return fmt.Errorf("exit node %q is not a node name, address, %s or %s", name, ExitNodeAuto, ExitNodeOff)
	case name == "default" || name == base.ExitNode:
		o.ExitNode = nil
	default:
		o.ExitNode = &name
	}
	return nil
}

func without(list []string, item string) []string {
	return slices.DeleteFunc(slices.Clone(list), func(s string) bool { return s == item })
}

// NixState renders p as the tailscale entry of generated/state.nix.
func NixState(p Policy) string {
	list := func(items []string) string {
		quoted := make([]string, len(items))
		for i, item := range items {
			quoted[i] = nixString(item)
		}
		return "[ " + strings.Join(append(quoted, ""), " ") + "]"
	}
	return fmt.Sprintf(
		"tailscale = { enable = true; homeSubnets = %s; trustedWifis = %s; exitNode = %s; siteRouterTrust = %t; siteRouterTargets = %s; };",
		list(p.HomeSubnets), list(p.TrustedWifis), nixString(p.ExitNode), p.SiteRouterTrust, list(p.SiteRouterTargets),
	)
}

func nixString(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "${", `\${`, "\n", `\n`)
	return `"` + r.Replace(s) + `"`
}
