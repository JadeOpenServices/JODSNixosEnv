package research

import "testing"

func TestAllowlistDoesNotAcceptLookalikes(t *testing.T) {
	b := Broker{AllowedHosts: []string{"nixos.org", "github.com"}}
	for _, host := range []string{"nixos.org", "search.nixos.org", "github.com"} {
		if !b.allowed(host) {
			t.Fatal(host)
		}
	}
	for _, host := range []string{"evilnixos.org", "github.com.evil.test", "localhost"} {
		if b.allowed(host) {
			t.Fatal(host)
		}
	}
}
