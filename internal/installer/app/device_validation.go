package app

import (
	"context"
	"fmt"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/oddcvalidation"
	"github.com/bakanura/gjallarOS/internal/installer/prompt"
)

var localValidationMatches = oddcvalidation.LocalValidationMatches

func enforceDeviceValidation(
	ctx context.Context,
	ui prompt.UI,
	user config.User,
	resolved oddc.Resolved,
	nixOSRelease string,
	gjallarOSRevision string,
) error {
	if resolved.Device.ID == "" {
		return nil
	}

	target := oddc.ValidationTarget{
		NixOSRelease:      nixOSRelease,
		GjallarOSRevision: gjallarOSRevision,
		DeviceID:          resolved.Device.ID,
		ODDCRevision:      resolved.Source.Revision,
	}

	if resolved.Device.Validation.Matches(target) {
		return nil
	}

	localMatch, err := localValidationMatches(ctx, target)
	if err != nil {
		return err
	}
	if localMatch {
		return nil
	}

	if user.AllowUnvalidatedDeviceProfile {
		return nil
	}

	if user.UnattendedInstall {
		return fmt.Errorf(
			"ODDC device profile %q is not validated for NixOS %q, GjallarOS %q, and ODDC revision %q; unattended installation requires allowUnvalidatedDeviceProfile=true",
			resolved.Device.ID,
			nixOSRelease,
			gjallarOSRevision,
			resolved.Source.Revision,
		)
	}

	approved, err := ui.Confirm(
		ctx,
		fmt.Sprintf(
			"WARNING: Device profile %q has not completed the full real-device validation gate for NixOS %s and the current GjallarOS/ODDC revisions. Continuing may cause hardware, graphics, tablet/sensor, or Secure Boot problems. Continue with this unvalidated device profile?",
			resolved.Device.ID,
			nixOSRelease,
		),
		false,
	)
	if err != nil {
		return err
	}
	if !approved {
		return fmt.Errorf(
			"installation cancelled because ODDC device profile %q is unvalidated for the current release",
			resolved.Device.ID,
		)
	}

	return nil
}
