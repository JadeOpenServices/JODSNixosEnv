package app

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/hardwarereconcile"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/oddchost"
	"github.com/bakanura/gjallarOS/internal/installer/prompt"
	"github.com/bakanura/gjallarOS/internal/usbtrust"
)

var detectInstallerHardware = discovery.DetectHardware

func reconcileInternalHardware(
	ctx context.Context,
	ui prompt.UI,
	resolved oddc.Resolved,
	hardware discovery.Hardware,
	host *oddc.HostOverlay,
	unattended bool,
	out io.Writer,
) (
	discovery.Hardware,
	*oddc.HostOverlay,
	bool,
	error,
) {
	hostChanged := false

	for {
		findings, err := hardwarereconcile.InternalUSB(
			resolved.Canonical.Resolved,
			hardware.DeviceInventory,
		)
		if err != nil {
			return hardware, host, hostChanged, fmt.Errorf(
				"reconcile ODDC internal hardware: %w",
				err,
			)
		}

		ambiguous := internalRolesByCode(
			findings,
			usbtrust.CodeInternalAmbiguous,
		)
		missing := internalRolesByCode(
			findings,
			usbtrust.CodeInternalMissing,
		)

		if len(ambiguous) == 0 && len(missing) == 0 {
			return hardware, host, hostChanged, nil
		}

		if unattended {
			roles := append(
				append([]string{}, ambiguous...),
				missing...,
			)
			roles = uniqueSortedRoles(roles)

			return hardware, host, hostChanged, fmt.Errorf(
				"ODDC internal hardware is unresolved during unattended installation: %s",
				strings.Join(roles, ", "),
			)
		}

		// Ambiguity means matching hardware is physically present but cannot be
		// identified safely. It must never be converted into a removal.
		if len(ambiguous) != 0 {
			var message strings.Builder

			message.WriteString(
				"ODDC found multiple USB devices matching internal hardware:\n\n",
			)
			for _, role := range ambiguous {
				fmt.Fprintf(&message, "- %s\n", role)
			}

			message.WriteString(
				"\nDisconnect any external USB device duplicating one of these identities.\n\n" +
					"Choose Yes when ready to rescan hardware. " +
					"Choose No to stop without changing ODDC.",
			)

			rescan, err := ui.Confirm(
				ctx,
				message.String(),
				false,
			)
			if err != nil {
				return hardware, host, hostChanged, err
			}

			if !rescan {
				return hardware, host, hostChanged, fmt.Errorf(
					"ODDC internal hardware remains ambiguous: %s",
					strings.Join(ambiguous, ", "),
				)
			}

			hardware = detectInstallerHardware("/sys")
			fmt.Fprintln(
				out,
				"Rescanned hardware for ambiguous ODDC internal components.",
			)
			continue
		}

		var message strings.Builder

		message.WriteString(
			"ODDC expects internal hardware that is not currently detected:\n\n",
		)
		for _, role := range missing {
			fmt.Fprintf(&message, "- %s\n", role)
		}

		message.WriteString(
			"\nEnable or reconnect any disabled internal component if it is still part of this machine.\n\n" +
				"Choose Yes when ready to rescan hardware. " +
				"Choose No to classify the missing component without rescanning.",
		)

		rescan, err := ui.Confirm(
			ctx,
			message.String(),
			false,
		)
		if err != nil {
			return hardware, host, hostChanged, err
		}

		if rescan {
			hardware = detectInstallerHardware("/sys")
			fmt.Fprintln(
				out,
				"Rescanned hardware for missing ODDC internal components.",
			)
			continue
		}

		for _, role := range missing {
			permanentlyRemoved, err := ui.Confirm(
				ctx,
				fmt.Sprintf(
					"Was %s permanently removed from this machine?\n\n"+
						"Choose Yes only if the component is intentionally no longer part of this machine. "+
						"Choose No if it is merely disabled, disconnected, unavailable, or temporarily missing.",
					role,
				),
				false,
			)
			if err != nil {
				return hardware, host, hostChanged, err
			}

			if !permanentlyRemoved {
				fmt.Fprintf(
					out,
					"WARN: %s remains in canonical ODDC and is temporarily absent.\n",
					role,
				)
				continue
			}

			if host == nil {
				staged, err := oddchost.New(resolved.ModelID)
				if err != nil {
					return hardware, nil, hostChanged, err
				}
				host = &staged
			}

			if err := oddchost.AddDeletion(
				host,
				role,
			); err != nil {
				return hardware, host, hostChanged, fmt.Errorf(
					"stage permanent ODDC hardware removal %s: %w",
					role,
					err,
				)
			}

			hostChanged = true

			fmt.Fprintf(
				out,
				"STAGED: permanent machine-local ODDC removal for %s.\n",
				role,
			)
		}

		return hardware, host, hostChanged, nil
	}
}

func internalRolesByCode(
	findings []usbtrust.Finding,
	code usbtrust.AuditCode,
) []string {
	roles := make([]string, 0)

	for _, finding := range findings {
		if finding.Code != code {
			continue
		}

		role := strings.TrimSpace(finding.Role)
		if role != "" {
			roles = append(roles, role)
		}
	}

	return uniqueSortedRoles(roles)
}

func uniqueSortedRoles(roles []string) []string {
	seen := map[string]struct{}{}

	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role != "" {
			seen[role] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for role := range seen {
		result = append(result, role)
	}

	sort.Strings(result)
	return result
}
