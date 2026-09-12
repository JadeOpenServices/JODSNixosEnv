package oddcvalidation

import (
	"fmt"
	"path/filepath"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/deviceprofile"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/sourcerevision"
)

type DeviceContext struct {
	User     config.User
	Hardware discovery.Hardware
	Resolved oddc.Resolved
	Revision string
}

type ConfigLoader func(path string) (config.User, error)
type RevisionResolver func(repo string, recovery bool) (string, error)
type HardwareDetector func(sysRoot string) discovery.Hardware

func LoadDeviceContext(repo string) (DeviceContext, error) {
	return loadDeviceContext(
		repo,
		config.Load,
		sourcerevision.Resolve,
		discovery.DetectHardware,
	)
}

func loadDeviceContext(
	repo string,
	loadConfig ConfigLoader,
	resolveRevision RevisionResolver,
	detectHardware HardwareDetector,
) (DeviceContext, error) {
	user, err := loadConfig(filepath.Join(repo, "user.config.json"))
	if err != nil {
		return DeviceContext{}, fmt.Errorf(
			"load device validation configuration: %w",
			err,
		)
	}

	revision, err := resolveRevision(repo, false)
	if err != nil {
		return DeviceContext{}, fmt.Errorf(
			"resolve GjallarOS source revision: %w",
			err,
		)
	}

	hardware := detectHardware("/sys")

	source := deviceprofile.CurrentEmbeddedSource(
		repo,
		revision,
	)

	resolved, err := deviceprofile.Resolve(source, hardware)
	if err != nil {
		return DeviceContext{}, err
	}

	return DeviceContext{
		User:     user,
		Hardware: hardware,
		Resolved: resolved,
		Revision: revision,
	}, nil
}

func (d DeviceContext) ValidationResults() []Result {
	return []Result{
		hardwareIdentityResult(d.User, d.Hardware),
		profilePropagationResult(d.User, d.Resolved),
	}
}
