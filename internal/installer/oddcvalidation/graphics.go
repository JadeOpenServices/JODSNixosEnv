package oddcvalidation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bakanura/gjallarOS/internal/hardware/graphics"
)

type graphicsSettings struct {
	Vendor          string `json:"vendor"`
	DeviceID        string `json:"deviceId"`
	Type            string `json:"type"`
	Compute         bool   `json:"compute"`
	BusID           string `json:"busId"`
	IntegratedBusID string `json:"integratedBusId"`
}

type GraphicsDetector func(context.Context) (graphics.Result, error)

const graphicsSettingsExpr = `
let
  flake = builtins.getFlake ("path:" + toString ./.);
  pkgs = flake.inputs.nixpkgs.legacyPackages.${builtins.currentSystem};
  settings = import ./generated/state.nix {
    inherit pkgs;
    inputs = flake.inputs;
  };
  resolved = flake.nixosConfigurations.${settings.hostname}.config.oddc.resolved;
  graphics = import (flake.inputs.oddc + "/nixos/lib/graphics.nix") {
    lib = flake.inputs.nixpkgs.lib;
  } resolved;
  selected = if graphics.discreteDriver != "" then graphics.discrete else graphics.integrated;
in {
  vendor = graphics.vendor;
  deviceId = selected.deviceId or "";
  type = graphics.type;
  compute = graphics.compute;
  busId = settings.graphicsBusId;
  integratedBusId = settings.graphicsIntegratedBusId;
}
`

func runtimeGraphicsResult(
	ctx context.Context,
	runner CommandRunner,
	detect GraphicsDetector,
	repo string,
) Result {
	output, err := runner(
		ctx,
		repo,
		"nix",
		"eval",
		"--json",
		"--impure",
		"--expr",
		graphicsSettingsExpr,
	)
	if err != nil {
		return Result{
			Gate:    GateRuntimeGraphics,
			Passed:  false,
			Details: fmt.Sprintf("evaluate resolved graphics state: %v", err),
		}
	}

	var expected graphicsSettings
	if err := json.Unmarshal(output, &expected); err != nil {
		return Result{
			Gate:    GateRuntimeGraphics,
			Passed:  false,
			Details: fmt.Sprintf("decode resolved graphics state: %v", err),
		}
	}

	current, err := detect(ctx)
	if err != nil {
		return Result{
			Gate:    GateRuntimeGraphics,
			Passed:  false,
			Details: fmt.Sprintf("detect runtime graphics: %v", err),
		}
	}

	if expected.Vendor != current.Vendor ||
		expected.DeviceID != current.DeviceID ||
		expected.Type != current.Type ||
		expected.Compute != current.Compute ||
		expected.BusID != current.BusID ||
		expected.IntegratedBusID != current.IntegratedBusID {
		return Result{
			Gate:   GateRuntimeGraphics,
			Passed: false,
			Details: fmt.Sprintf(
				"expected=%+v runtime=%+v",
				expected,
				current,
			),
		}
	}

	return Result{
		Gate:   GateRuntimeGraphics,
		Passed: true,
	}
}
