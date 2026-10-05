package app

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/prompt"
)

func collectTailscaleAnswers(t *testing.T, answers ...string) (*config.TailscaleIntent, string) {
	t.Helper()
	var output bytes.Buffer
	user := config.User{}
	ui := prompt.New(strings.NewReader(strings.Join(answers, "\n")+"\n"), &output)
	if err := collectTailscale(context.Background(), ui, &user); err != nil {
		t.Fatal(err)
	}
	return user.Tailscale, output.String()
}

func TestCollectTailscaleHomeWifisAndSiteRouter(t *testing.T) {
	got, _ := collectTailscaleAnswers(t,
		"", // enable
		"", // home subnets: default
		"y",
		"home-router",
		"bakasifu-5Ghz, bakasifu-2.4Ghz,fizzlipuzzli",
		"", // site router trust
		"", // targets: default
	)
	want := config.TailscaleIntent{
		Enable:            true,
		HomeSubnets:       []string{"192.168.8.0/24"},
		TrustedWifis:      []string{"bakasifu-5Ghz", "bakasifu-2.4Ghz", "fizzlipuzzli"},
		ExitNode:          "home-router",
		SiteRouterTrust:   true,
		SiteRouterTargets: []string{"192.168.8.1:53"},
	}
	if got.Enable != want.Enable || got.ExitNode != want.ExitNode || !got.SiteRouterTrust ||
		!slices.Equal(got.HomeSubnets, want.HomeSubnets) ||
		!slices.Equal(got.TrustedWifis, want.TrustedWifis) ||
		!slices.Equal(got.SiteRouterTargets, want.SiteRouterTargets) {
		t.Fatalf("tailscale = %#v", got)
	}
}

func TestCollectTailscaleReasksInvalidAnswers(t *testing.T) {
	got, out := collectTailscaleAnswers(t,
		"y",
		"192.168.8.0", // not a CIDR
		"10.0.0.0/24",
		"y",
		"", // exit node is required
		"exit-1",
		"",
		"y",
		"router", // no port
		"10.0.0.1:443",
	)
	if !strings.Contains(out, "is not a CIDR subnet") || !strings.Contains(out, "is not host:port") {
		t.Fatalf("missing validation messages:\n%s", out)
	}
	if got.ExitNode != "exit-1" || !slices.Equal(got.HomeSubnets, []string{"10.0.0.0/24"}) ||
		!slices.Equal(got.SiteRouterTargets, []string{"10.0.0.1:443"}) || len(got.TrustedWifis) != 0 {
		t.Fatalf("tailscale = %#v", got)
	}
}

func TestCollectTailscaleWithoutVPNKeepsHomeBypassOnly(t *testing.T) {
	got, _ := collectTailscaleAnswers(t, "y", "", "n")
	if !got.Enable || got.ExitNode != "" || got.SiteRouterTrust || len(got.TrustedWifis) != 0 {
		t.Fatalf("tailscale = %#v", got)
	}
}

func TestCollectTailscaleDisabled(t *testing.T) {
	got, _ := collectTailscaleAnswers(t, "n")
	if got == nil || got.Enable {
		t.Fatalf("tailscale = %#v", got)
	}
}
