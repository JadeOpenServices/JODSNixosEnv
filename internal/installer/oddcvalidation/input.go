package oddcvalidation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bakanura/gjallarOS/internal/installer/discovery"
)

type inputSettings struct {
	Touchscreen bool `json:"touchscreen"`
	PenTablet   bool `json:"penTablet"`
}

func runtimeInputResult(
	ctx context.Context,
	runner CommandRunner,
	detect HardwareDetector,
	repo string,
) Result {
	output, err := runner(
		ctx,
		repo,
		"nix", "eval", "--json", "--impure", "--expr",
		`let
		  flake = builtins.getFlake ("path:" + toString ./.);
		  pkgs = flake.inputs.nixpkgs.legacyPackages.${builtins.currentSystem};
		  settings = import ./settings.nix { inherit pkgs; inputs = flake.inputs; };
		in {
		  touchscreen = settings.touchscreenEnable;
		  penTablet = settings.penTabletEnable;
		}`,
	)
	if err != nil {
		return Result{
			Gate:    GateRuntimeSensorsTablet,
			Details: fmt.Sprintf("evaluate generated input state: %v", err),
		}
	}

	var expected inputSettings
	if err := json.Unmarshal(output, &expected); err != nil {
		return Result{
			Gate:    GateRuntimeSensorsTablet,
			Details: fmt.Sprintf("decode generated input state: %v", err),
		}
	}

	current := detect("/sys")

	if expected.Touchscreen != current.Touchscreen ||
		expected.PenTablet != current.PenTablet {
		return Result{
			Gate: GateRuntimeSensorsTablet,
			Details: fmt.Sprintf(
				"generated touchscreen=%t penTablet=%t; runtime touchscreen=%t penTablet=%t",
				expected.Touchscreen,
				expected.PenTablet,
				current.Touchscreen,
				current.PenTablet,
			),
		}
	}

	return Result{
		Gate:   GateRuntimeSensorsTablet,
		Passed: true,
	}
}

var _ = discovery.Hardware{}
