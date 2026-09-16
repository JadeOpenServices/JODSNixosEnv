package oddcvalidation

import "fmt"

type Gate string

const (
	GateFocusedODDCTests     Gate = "focused-oddc-tests"
	GateGoTests              Gate = "go-tests"
	GateInstallerCheck       Gate = "installer-check"
	GateFlakeCheck           Gate = "flake-check"
	GateDeviceEvaluation     Gate = "device-evaluation"
	GateRealMachineRebuild   Gate = "real-machine-rebuild"
	GateRuntimeGraphics      Gate = "runtime-graphics"
	GateRuntimeSensorsTablet Gate = "runtime-sensors-tablet"
	GateSecureBootPolicy     Gate = "secure-boot-policy"
)

var RequiredGates = []Gate{
	GateFocusedODDCTests,
	GateGoTests,
	GateInstallerCheck,
	GateFlakeCheck,
	GateDeviceEvaluation,
	GateRealMachineRebuild,
	GateRuntimeGraphics,
	GateRuntimeSensorsTablet,
	GateSecureBootPolicy,
}

type Result struct {
	Gate    Gate
	Passed  bool
	Details string
}

type Report struct {
	Results []Result
}

func (r Report) Complete() error {
	seen := make(map[Gate]Result, len(r.Results))
	for _, result := range r.Results {
		if _, exists := seen[result.Gate]; exists {
			return fmt.Errorf("duplicate ODDC validation gate %q", result.Gate)
		}
		seen[result.Gate] = result
	}

	for _, gate := range RequiredGates {
		result, ok := seen[gate]
		if !ok {
			return fmt.Errorf("ODDC validation gate %q was not executed", gate)
		}
		if !result.Passed {
			if result.Details != "" {
				return fmt.Errorf(
					"ODDC validation gate %q failed: %s",
					gate,
					result.Details,
				)
			}
			return fmt.Errorf("ODDC validation gate %q failed", gate)
		}
	}

	return nil
}
