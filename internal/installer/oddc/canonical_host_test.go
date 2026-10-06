package oddc_test

import (
	"sort"
	"testing"

	portable "github.com/JadeOpenServices/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/oddc/oddctest"
)

func TestEmbeddedSourceAppliesHostOverlayWithoutChangingSourceMetadata(
	t *testing.T,
) {
	for _, answer := range oddctest.Answers(t) {
		source := oddc.EmbeddedSource{
			Root:       answer.Root,
			Repository: "embedded:test",
			Revision:   "source-revision",
			Integrity:  "source-integrity",
		}
		identity := oddc.Identity(answer.Identity)

		plain, err := source.Resolve(identity)
		if err != nil {
			t.Fatal(err)
		}

		hardware, _ := plain.Canonical.Resolved["hardware"].(map[string]any)
		if len(hardware) == 0 {
			t.Fatalf("%s resolves no hardware", answer.Model)
		}
		keys := make([]string, 0, len(hardware))
		for key := range hardware {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		deleted := "hardware." + keys[0]

		host := oddc.HostOverlay{
			APIVersion:  portable.EntityAPIVersion,
			ID:          "host/test",
			Kind:        "host",
			TargetModel: answer.Model,
			Overrides: map[string]any{
				"hardware": map[string]any{
					keys[0]: map[string]any{"$delete": true},
				},
			},
		}

		resolved, err := source.ResolveWithHost(identity, []oddc.HostOverlay{host})
		if err != nil {
			t.Fatal(err)
		}

		if _, exists := portable.Lookup(resolved.Canonical.Resolved, deleted); exists {
			t.Fatalf("%s: host deletion of %s was not applied", answer.Model, deleted)
		}

		if resolved.Source.Repository != "embedded:test" ||
			resolved.Source.Revision != "source-revision" ||
			resolved.Source.Integrity != "source-integrity" {
			t.Fatalf("host overlay changed source metadata: %+v", resolved.Source)
		}

		if source.Metadata().Repository != "embedded:test" {
			t.Fatalf("embedded source metadata changed: %+v", source.Metadata())
		}
	}
}
