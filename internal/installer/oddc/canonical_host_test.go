package oddc

import (
	"os"
	"path/filepath"
	"testing"

	portable "github.com/bakanura/gjallarOS/pkg/oddc"
)

func TestEmbeddedSourceAppliesHostOverlayWithoutChangingSourceMetadata(
	t *testing.T,
) {
	root := t.TempDir()

	modelPath := filepath.Join(
		root,
		"catalog",
		"entities",
		"models",
		"test",
		"test-laptop.json",
	)

	if err := os.MkdirAll(filepath.Dir(modelPath), 0755); err != nil {
		t.Fatal(err)
	}

	model := `{
  "apiVersion": "oddc.openjade.de/v2",
  "kind": "DeviceModel",
  "metadata": {
    "id": "model/test/test-laptop",
    "name": "Test Laptop"
  },
  "data": {
    "class": {
      "formFactor": "laptop"
    },
    "identity": {
      "dmi": {
        "systemVendor": {
          "hp": "HP"
        },
        "productName": {
          "test-laptop": "Test Laptop"
        }
      }
    },
    "hardware": {
      "security": {
        "fingerprint": {
          "primary": {
            "bus": "usb",
            "attachment": "internal",
            "deviceId": "1234:5678"
          }
        }
      }
    }
  }
}
`

	if err := os.WriteFile(modelPath, []byte(model), 0644); err != nil {
		t.Fatal(err)
	}

	source := EmbeddedSource{
		Root:       root,
		Repository: "embedded:test",
		Revision:   "source-revision",
		Integrity:  "source-integrity",
	}

	host := HostOverlay{
		APIVersion:  portable.EntityAPIVersion,
		ID:          "host/test",
		Kind:        "host",
		TargetModel: "model/test/test-laptop",
		Overrides: map[string]any{
			"hardware": map[string]any{
				"security": map[string]any{
					"fingerprint": map[string]any{
						"primary": map[string]any{
							"$delete": true,
						},
					},
				},
			},
		},
	}

	resolved, err := source.ResolveWithHost(
		Identity{
			FormFactor:  "laptop",
			SysVendor:   "HP",
			ProductName: "Test Laptop",
		},
		[]HostOverlay{host},
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, exists := portable.Lookup(
		resolved.Canonical.Resolved,
		"hardware.security.fingerprint.primary",
	); exists {
		t.Fatal("host deletion was not applied")
	}

	if resolved.Source.Repository != "embedded:test" ||
		resolved.Source.Revision != "source-revision" ||
		resolved.Source.Integrity != "source-integrity" {
		t.Fatalf(
			"host overlay changed source metadata: %+v",
			resolved.Source,
		)
	}

	if source.Metadata().Repository != "embedded:test" {
		t.Fatalf(
			"embedded source metadata changed: %+v",
			source.Metadata(),
		)
	}
}
