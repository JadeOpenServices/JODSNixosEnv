package oddc_test

import (
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/oddc/oddctest"
)

// Every catalog model yields valid policies; a model without a Secure Boot
// firmware policy fails closed with a reason.
func TestEveryModelResolvesPolicies(t *testing.T) {
	for _, answer := range oddctest.Answers(t) {
		source := oddc.EmbeddedSource{Root: answer.Root, Repository: "embedded:oddc"}

		resolved, err := source.Resolve(oddc.Identity(answer.Identity))
		if err != nil {
			t.Fatal(err)
		}
		if resolved.ModelID != answer.Model {
			t.Fatalf("%s identity resolves %q", answer.Model, resolved.ModelID)
		}

		firmware, err := oddc.ResolveSecureBootFirmwarePolicy(resolved)
		if err != nil {
			t.Fatalf("%s firmware policy: %v", answer.Model, err)
		}
		if !firmware.Policy.Supported &&
			strings.TrimSpace(firmware.Policy.UnsupportedReason) == "" {
			t.Errorf("%s: unsupported firmware policy without a reason", answer.Model)
		}
		if firmware.Policy.Supported && firmware.SourceEntity == "" {
			t.Errorf("%s: supported firmware policy without a source entity", answer.Model)
		}

		if _, err := oddc.ResolveGraphicsPolicy(resolved); err != nil {
			t.Errorf("%s graphics policy: %v", answer.Model, err)
		}
	}
}
