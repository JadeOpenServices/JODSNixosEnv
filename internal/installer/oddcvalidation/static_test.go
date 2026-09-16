package oddcvalidation

import (
	"context"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
)

func TestRunStaticProducesExpectedGateOrder(t *testing.T) {
	runner := func(
		ctx context.Context,
		dir string,
		name string,
		args ...string,
	) ([]byte, error) {
		return nil, nil
	}

	report := runStatic(
		context.Background(),
		t.TempDir(),
		runner,
		func(path string) (config.User, error) {
			return config.User{Hostname: "gjallarOS"}, nil
		},
	)

	want := []Gate{
		GateFocusedODDCTests,
		GateGoTests,
		GateInstallerCheck,
		GateFlakeCheck,
		GateDeviceEvaluation,
	}

	if len(report.Results) != len(want) {
		t.Fatalf("results=%d want %d", len(report.Results), len(want))
	}

	for i, gate := range want {
		if report.Results[i].Gate != gate {
			t.Fatalf(
				"result[%d].Gate=%q want %q",
				i,
				report.Results[i].Gate,
				gate,
			)
		}
	}
}
