package recoveryresize

import (
	"errors"
	"fmt"
	"strings"
)

const ManifestVersion = 1

type Manifest struct {
	Version int `json:"version"`

	Nonce string `json:"nonce"`
	Stage string `json:"stage"`

	DiskGUID      string `json:"disk_guid"`
	DiskSizeBytes uint64 `json:"disk_size_bytes"`

	RootPARTUUID   string `json:"root_partuuid"`
	RootStartBytes uint64 `json:"root_start_bytes"`

	ExpectedCurrentRootEndBytes uint64 `json:"expected_current_root_end_bytes"`
	ExpectedNewRootEndBytes     uint64 `json:"expected_new_root_end_bytes"`

	LUKSUUID  string `json:"luks_uuid"`
	BtrfsUUID string `json:"btrfs_uuid"`

	RecoveryBytes uint64 `json:"recovery_bytes"`
}

func ManifestFromPlan(
	topology Topology,
	plan Plan,
	nonce string,
) (Manifest, error) {
	nonce = strings.TrimSpace(nonce)
	if nonce == "" {
		return Manifest{}, errors.New("manifest nonce is required")
	}

	return Manifest{
		Version: ManifestVersion,

		Nonce: nonce,
		Stage: StagePreparingResize,

		DiskGUID:      topology.DiskGUID,
		DiskSizeBytes: topology.DiskSizeBytes,

		RootPARTUUID:   topology.RootPARTUUID,
		RootStartBytes: topology.RootStartBytes,

		ExpectedCurrentRootEndBytes: topology.RootEndBytes,
		ExpectedNewRootEndBytes:     plan.NewRootEndBytes,

		LUKSUUID:  topology.LUKSUUID,
		BtrfsUUID: topology.BtrfsUUID,

		RecoveryBytes: plan.RecoveryBytes,
	}, nil
}

// ValidateResume verifies only identity and state.
//
// Authentication of the serialized manifest belongs to resizebootstrap.
// A manifest is never authority for geometry by itself: callers must
// rediscover Topology and BuildPlan again before any mutation.
func ValidateResume(
	current Topology,
	manifest Manifest,
	expectedStage string,
) error {
	if manifest.Version != ManifestVersion {
		return fmt.Errorf(
			"unsupported recovery resize manifest version %d",
			manifest.Version,
		)
	}
	if strings.TrimSpace(manifest.Nonce) == "" {
		return errors.New("manifest nonce is missing")
	}
	if manifest.Stage != expectedStage {
		return fmt.Errorf(
			"stale or inconsistent resume stage: manifest=%q expected=%q",
			manifest.Stage,
			expectedStage,
		)
	}

	checks := []struct {
		name string
		got  string
		want string
	}{
		{"disk GUID", current.DiskGUID, manifest.DiskGUID},
		{"root PARTUUID", current.RootPARTUUID, manifest.RootPARTUUID},
		{"LUKS UUID", current.LUKSUUID, manifest.LUKSUUID},
		{"Btrfs UUID", current.BtrfsUUID, manifest.BtrfsUUID},
	}
	for _, check := range checks {
		if !strings.EqualFold(
			strings.TrimSpace(check.got),
			strings.TrimSpace(check.want),
		) {
			return fmt.Errorf(
				"%s mismatch: current=%q expected=%q",
				check.name,
				check.got,
				check.want,
			)
		}
	}

	if current.DiskSizeBytes != manifest.DiskSizeBytes {
		return fmt.Errorf(
			"disk size mismatch: current=%d expected=%d",
			current.DiskSizeBytes,
			manifest.DiskSizeBytes,
		)
	}
	if current.RootStartBytes != manifest.RootStartBytes {
		return fmt.Errorf(
			"root start mismatch: current=%d expected=%d",
			current.RootStartBytes,
			manifest.RootStartBytes,
		)
	}
	if current.RootEndBytes !=
		manifest.ExpectedCurrentRootEndBytes {
		return fmt.Errorf(
			"root end mismatch: current=%d expected=%d",
			current.RootEndBytes,
			manifest.ExpectedCurrentRootEndBytes,
		)
	}

	return ValidateTopology(current)
}
