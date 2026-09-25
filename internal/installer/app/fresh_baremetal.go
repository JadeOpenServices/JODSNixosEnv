package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bakanura/gjallarOS/internal/installer/baremetalinstall"
	"github.com/bakanura/gjallarOS/internal/installer/credential"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/diskplan"
	"github.com/bakanura/gjallarOS/internal/installer/freshdiskplan"
	"github.com/bakanura/gjallarOS/internal/installer/freshgpt"
	"github.com/bakanura/gjallarOS/internal/installer/hardwareconfig"
	"github.com/bakanura/gjallarOS/internal/installer/installconfirm"
	"github.com/bakanura/gjallarOS/internal/installer/mounttree"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/prompt"
	"github.com/bakanura/gjallarOS/internal/installer/rootprovision"
	"github.com/bakanura/gjallarOS/internal/installer/targetdisk"
)

type freshBareMetalResult struct {
	TargetDisk        string
	RootPARTUUID      string
	LUKSUUID          string
	RecoveryPartition string
}

func runFreshBareMetal(
	ctx context.Context,
	ui prompt.UI,
	repo string,
	targetPath string,
	hostname string,
	hardware discovery.Hardware,
	resolvedDevice oddc.Resolved,
	recovery bool,
	enableRecovery bool,
	passwordFiles []string,
	hostOverlay *oddc.HostOverlay,
	out io.Writer,
) (freshBareMetalResult, error) {
	if out == nil {
		return freshBareMetalResult{}, fmt.Errorf(
			"fresh installation output writer is required",
		)
	}

	targetPath = strings.TrimSpace(targetPath)

	fmt.Fprintln(out, "STAGE: validating fresh installation target")

	observed, err := targetdisk.Validate(
		ctx,
		targetdisk.Input{
			Path: targetPath,
		},
	)
	if err != nil {
		return freshBareMetalResult{}, fmt.Errorf(
			"validate fresh target disk: %w",
			err,
		)
	}

	plan, err := freshdiskplan.Build(
		freshdiskplan.Input{
			Observed:       observed,
			EnableRecovery: enableRecovery,
		},
	)
	if err != nil {
		return freshBareMetalResult{}, fmt.Errorf(
			"build canonical fresh disk plan: %w",
			err,
		)
	}

	// Never infer unattended authorization from managed-device, recovery, JODS,
	// preset, or any other feature state. Interactive fresh installs require
	// the exact destructive phrase here.
	if err := installconfirm.Confirm(
		ctx,
		ui,
		out,
		plan,
		observed,
		installconfirm.Options{
			Unattended: false,
		},
	); err != nil {
		return freshBareMetalResult{}, fmt.Errorf(
			"fresh disk authorization: %w",
			err,
		)
	}

	fmt.Fprintln(out, "STAGE: creating fresh canonical partition table")

	if _, err := freshgpt.Provision(
		ctx,
		freshgpt.Input{
			Plan: plan,
			Out:  out,
		},
	); err != nil {
		return freshBareMetalResult{}, fmt.Errorf(
			"provision fresh GPT: %w",
			err,
		)
	}

	passphrase, err := readFreshLUKSPassphrase(out)
	if err != nil {
		return freshBareMetalResult{}, err
	}
	secret := []byte(passphrase)
	passphrase = ""
	defer zero(secret)

	fmt.Fprintln(out, "STAGE: provisioning encrypted Btrfs root")

	rootResult, err := rootprovision.Provision(
		ctx,
		rootprovision.Input{
			Plan:       plan,
			Passphrase: secret,
			Out:        out,
		},
	)
	if err != nil {
		return freshBareMetalResult{}, fmt.Errorf(
			"provision encrypted root: %w",
			err,
		)
	}

	// Fresh canonical storage has exactly one owner. mounttree prepares the
	// ESP, encrypted Btrfs root mount, and the exact 12 GiB JODS-RECOVERY
	// partition described by the canonical disk plan. Recovery artifact
	// population is a later trust operation and must not own filesystem layout.
	fmt.Fprintln(out, "STAGE: preparing fresh target mount tree")

	if _, err := mounttree.Prepare(
		ctx,
		mounttree.Input{
			Plan: plan,
			Out:  out,
		},
	); err != nil {
		return freshBareMetalResult{}, fmt.Errorf(
			"prepare fresh target mount tree: %w",
			err,
		)
	}

	fmt.Fprintln(
		out,
		"STAGE: materializing machine-local device profile into target",
	)

	if err := materializeODDCCapsule(
		repo,
		filepath.Join(
			rootResult.MountPoint,
			"var",
			"lib",
			"gjallarOS",
			"device-profile",
		),
		hardware,
		resolvedDevice,
		recovery,
		"initial",
	); err != nil {
		return freshBareMetalResult{}, fmt.Errorf(
			"materialize fresh target device profile: %w",
			err,
		)
	}

	fmt.Fprintln(out, "STAGE: staging account password hashes into target")

	if err := stageFreshPasswordFiles(passwordFiles); err != nil {
		return freshBareMetalResult{}, fmt.Errorf(
			"stage fresh target password hashes: %w",
			err,
		)
	}

	hardwarePath := hardwareconfig.Target(repo)

	fmt.Fprintln(out, "STAGE: generating target hardware configuration")

	if _, err := hardwareconfig.GenerateTarget(
		ctx,
		repo,
		hardwarePath,
		time.Now(),
	); err != nil {
		return freshBareMetalResult{}, fmt.Errorf(
			"generate fresh target hardware configuration: %w",
			err,
		)
	}

	fmt.Fprintln(out, "STAGE: installing GjallarOS into prepared target")

	if _, err := baremetalinstall.Install(
		ctx,
		baremetalinstall.Input{
			Repo:     repo,
			Hostname: hostname,
			Out:      out,
		},
	); err != nil {
		return freshBareMetalResult{}, fmt.Errorf(
			"install fresh GjallarOS: %w",
			err,
		)
	}

	if err := saveODDCHostOverlay(
		ctx,
		rootResult.MountPoint,
		hostOverlay,
	); err != nil {
		return freshBareMetalResult{}, fmt.Errorf(
			"commit fresh target ODDC host state: %w",
			err,
		)
	}

	if hostOverlay != nil {
		fmt.Fprintln(
			out,
			"PASS: machine-local ODDC host state committed to fresh target.",
		)
	}

	result := freshBareMetalResult{
		TargetDisk:   plan.TargetDisk.Path,
		RootPARTUUID: plan.Root.Partition.PARTUUID,
		LUKSUUID:     rootResult.LUKSUUID,
	}

	if plan.Recovery != nil {
		// Return stable identity, never /dev/nvmeXpN naming as authority.
		result.RecoveryPartition =
			"/dev/disk/by-partuuid/" +
				strings.ToLower(plan.Recovery.PARTUUID)
	}

	fmt.Fprintln(
		out,
		"PASS: canonical fresh bare-metal installation completed",
	)

	return result, nil
}

func stageFreshPasswordFiles(paths []string) error {
	const sourceRoot = "/var/lib/gjallarOS/passwords"
	const targetRoot = "/mnt/var/lib/gjallarOS/passwords"

	if err := os.MkdirAll(targetRoot, 0700); err != nil {
		return fmt.Errorf("create target password directory: %w", err)
	}

	seen := map[string]bool{}

	for _, source := range paths {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}

		source = filepath.Clean(source)

		rel, err := filepath.Rel(sourceRoot, source)
		if err != nil {
			return fmt.Errorf("resolve password hash path %q: %w", source, err)
		}

		if rel == "." ||
			filepath.IsAbs(rel) ||
			strings.HasPrefix(rel, ".."+string(filepath.Separator)) ||
			strings.Contains(rel, string(filepath.Separator)) {
			return fmt.Errorf(
				"password hash is outside canonical runtime directory: %s",
				source,
			)
		}

		if seen[rel] {
			continue
		}
		seen[rel] = true

		info, err := os.Lstat(source)
		if err != nil {
			return fmt.Errorf("inspect password hash %s: %w", source, err)
		}

		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf(
				"password hash must be a regular non-symlink file: %s",
				source,
			)
		}

		if info.Mode().Perm()&0077 != 0 {
			return fmt.Errorf(
				"password hash has unsafe permissions %04o: %s",
				info.Mode().Perm(),
				source,
			)
		}

		data, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("read password hash %s: %w", source, err)
		}

		if len(strings.TrimSpace(string(data))) == 0 {
			return fmt.Errorf("password hash is empty: %s", source)
		}

		destination := filepath.Join(targetRoot, rel)

		tmp, err := os.CreateTemp(
			targetRoot,
			"."+rel+".tmp-*",
		)
		if err != nil {
			return fmt.Errorf("create target password temp file: %w", err)
		}

		tmpName := tmp.Name()
		ok := false

		func() {
			defer func() {
				if !ok {
					_ = os.Remove(tmpName)
				}
			}()

			if err = tmp.Chmod(0600); err != nil {
				_ = tmp.Close()
				return
			}

			if _, err = tmp.Write(data); err != nil {
				_ = tmp.Close()
				return
			}

			if err = tmp.Sync(); err != nil {
				_ = tmp.Close()
				return
			}

			if err = tmp.Close(); err != nil {
				return
			}

			err = os.Rename(tmpName, destination)
			if err == nil {
				ok = true
			}
		}()

		for i := range data {
			data[i] = 0
		}

		if err != nil {
			return fmt.Errorf(
				"install target password hash %s: %w",
				destination,
				err,
			)
		}
	}

	return nil
}

func readFreshLUKSPassphrase(out io.Writer) (string, error) {
	// The encryption secret must not use prompt.UI.Value because that would
	// echo/log normal text input. Read directly from the controlling terminal
	// with echo disabled.
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf(
			"open controlling terminal for LUKS passphrase: %w",
			err,
		)
	}
	defer tty.Close()

	passphrase, err := credential.ReadConfirmedPassword(
		tty,
		out,
		"encrypted GjallarOS root",
	)
	if err != nil {
		return "", fmt.Errorf(
			"read LUKS passphrase: %w",
			err,
		)
	}

	if passphrase == "" {
		return "", fmt.Errorf("LUKS passphrase must not be empty")
	}

	return passphrase, nil
}

func zero(secret []byte) {
	for i := range secret {
		secret[i] = 0
	}
}

// Compile-time contract: fresh canonical storage is a diskplan.Plan and does
// not gain another parallel storage-model representation.
var _ diskplan.Plan
