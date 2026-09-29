package recoveryresize

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type DiscoveryInput struct {
	RootMountpoint string

	// These are locators from the already validated installer disk plan.
	// Runtime identity is independently rediscovered and checked below.
	ExpectedDiskPath          string
	ExpectedRootPartitionPath string
	ExpectedMappingPath       string
}

type luksJSONMetadata struct {
	Segments map[string]struct {
		Type   string `json:"type"`
		Offset string `json:"offset"`
		Size   string `json:"size"`
	} `json:"segments"`
}

var (
	cryptStatusDevicePattern = regexp.MustCompile(
		`(?m)^\s*device:\s*(\S+)\s*$`,
	)

	btrfsHeaderPattern = regexp.MustCompile(
		`(?m)\buuid:\s*([0-9A-Fa-f-]+)\s*$`,
	)

	btrfsDevicePattern = regexp.MustCompile(
		`(?m)^\s*devid\s+([0-9]+)\s+size\s+([0-9]+)\s+used\s+([0-9]+)\s+path\s+(\S+)\s*$`,
	)
)

func DiscoverTopology(
	ctx context.Context,
	runner OutputRunner,
	input DiscoveryInput,
) (Topology, error) {
	mountpoint := strings.TrimSpace(input.RootMountpoint)
	if mountpoint == "" {
		return Topology{}, errors.New("root mountpoint is required")
	}
	if !filepath.IsAbs(mountpoint) {
		return Topology{}, errors.New("root mountpoint must be absolute")
	}

	diskPath := strings.TrimSpace(input.ExpectedDiskPath)
	rootPart := strings.TrimSpace(input.ExpectedRootPartitionPath)
	expectedMapping := strings.TrimSpace(input.ExpectedMappingPath)

	for name, value := range map[string]string{
		"expected disk path":           diskPath,
		"expected root partition path": rootPart,
		"expected mapping path":        expectedMapping,
	} {
		if value == "" {
			return Topology{}, fmt.Errorf("%s is required", name)
		}
		if !filepath.IsAbs(value) {
			return Topology{}, fmt.Errorf("%s must be absolute", name)
		}
	}

	mountRaw, err := runner.Output(
		ctx,
		"findmnt",
		"-nvro",
		"SOURCE,FSTYPE,OPTIONS",
		"--target",
		mountpoint,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"discover mounted root topology: %w",
			err,
		)
	}

	source, fs, options, err := parseFindmnt(string(mountRaw))
	if err != nil {
		return Topology{}, err
	}

	if filepath.Clean(source) != filepath.Clean(expectedMapping) {
		return Topology{}, fmt.Errorf(
			"root mapping mismatch: mounted=%q expected=%q",
			source,
			expectedMapping,
		)
	}

	if err := RequireBtrfsFilesystem(fs); err != nil {
		return Topology{}, err
	}

	rootMountedRW := mountHasOption(options, "rw")
	if !rootMountedRW {
		return Topology{}, errors.New(
			"Btrfs root is not mounted read-write",
		)
	}

	mappingName := filepath.Base(source)
	if mappingName == "." || mappingName == "/" || mappingName == "" {
		return Topology{}, errors.New("invalid root mapping name")
	}

	cryptStatus, err := runner.Output(
		ctx,
		"cryptsetup",
		"status",
		mappingName,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"inspect active LUKS mapping %s: %w",
			mappingName,
			err,
		)
	}

	backingDevice, err := parseCryptStatusDevice(string(cryptStatus))
	if err != nil {
		return Topology{}, err
	}
	if filepath.Clean(backingDevice) != filepath.Clean(rootPart) {
		return Topology{}, fmt.Errorf(
			"LUKS backing-device mismatch: current=%q expected=%q",
			backingDevice,
			rootPart,
		)
	}

	partType, err := outputTrim(
		ctx,
		runner,
		"lsblk",
		"-dnro",
		"TYPE",
		rootPart,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"inspect root partition type: %w",
			err,
		)
	}
	if partType != "part" {
		return Topology{}, fmt.Errorf(
			"expected root device %s is type %q, not partition",
			rootPart,
			partType,
		)
	}

	diskType, err := outputTrim(
		ctx,
		runner,
		"lsblk",
		"-dnro",
		"TYPE",
		diskPath,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"inspect target disk type: %w",
			err,
		)
	}
	switch diskType {
	case "disk":
		// Normal bare-metal installer target.
	case "loop":
		// Loop-backed block devices are supported only as storage
		// containers by this lower-level topology layer. The production
		// GJAL-31 provisioning boundary independently requires TYPE=disk,
		// so accepting loop here enables real integration testing without
		// making loop devices valid installer targets.
	default:
		return Topology{}, fmt.Errorf(
			"expected storage container %s is unsupported block type %q",
			diskPath,
			diskType,
		)
	}

	parentName, err := outputTrim(
		ctx,
		runner,
		"lsblk",
		"-dnro",
		"PKNAME",
		rootPart,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"discover root partition parent disk: %w",
			err,
		)
	}

	if parentName == "" {
		return Topology{}, errors.New(
			"root partition has no physical parent disk",
		)
	}

	parentPath := parentName
	if !filepath.IsAbs(parentPath) {
		parentPath = filepath.Join("/dev", parentName)
	}

	if filepath.Clean(parentPath) != filepath.Clean(diskPath) {
		return Topology{}, fmt.Errorf(
			"root partition parent mismatch: current=%q expected=%q",
			parentPath,
			diskPath,
		)
	}

	sectorSize, err := outputUint(
		ctx,
		runner,
		"blockdev",
		"--getss",
		diskPath,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"read target logical sector size: %w",
			err,
		)
	}

	diskSize, err := outputUint(
		ctx,
		runner,
		"blockdev",
		"--getsize64",
		diskPath,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"read target disk size: %w",
			err,
		)
	}

	rootSize, err := outputUint(
		ctx,
		runner,
		"blockdev",
		"--getsize64",
		rootPart,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"read root partition size: %w",
			err,
		)
	}

	startSectors, err := outputUint(
		ctx,
		runner,
		"lsblk",
		"-dnro",
		"START",
		rootPart,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"read root partition start: %w",
			err,
		)
	}

	rootStart, err := multiply(startSectors, sectorSize)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"calculate root byte offset: %w",
			err,
		)
	}

	rootEnd, err := add(rootStart, rootSize)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"calculate root end: %w",
			err,
		)
	}

	diskGUID, err := outputTrim(
		ctx,
		runner,
		"lsblk",
		"-dnro",
		"PTUUID",
		diskPath,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"read GPT disk GUID: %w",
			err,
		)
	}
	if diskGUID == "" {
		return Topology{}, errors.New(
			"target disk GPT GUID is empty",
		)
	}

	rootPartNumber, err := outputUint(
		ctx,
		runner,
		"lsblk",
		"-dnro",
		"PARTN",
		rootPart,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"read root partition number: %w",
			err,
		)
	}
	if rootPartNumber == 0 {
		return Topology{}, errors.New(
			"root partition number is empty",
		)
	}

	rootTypeGUID, err := outputTrim(
		ctx,
		runner,
		"lsblk",
		"-dnro",
		"PARTTYPE",
		rootPart,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"read root partition type GUID: %w",
			err,
		)
	}
	if rootTypeGUID == "" {
		return Topology{}, errors.New(
			"root partition type GUID is empty",
		)
	}

	rootPARTUUID, err := outputTrim(
		ctx,
		runner,
		"lsblk",
		"-dnro",
		"PARTUUID",
		rootPart,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"read root PARTUUID: %w",
			err,
		)
	}
	if rootPARTUUID == "" {
		return Topology{}, errors.New(
			"root PARTUUID is empty",
		)
	}

	luksUUID, err := outputTrim(
		ctx,
		runner,
		"cryptsetup",
		"luksUUID",
		rootPart,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"read LUKS UUID: %w",
			err,
		)
	}
	if luksUUID == "" {
		return Topology{}, errors.New("LUKS UUID is empty")
	}

	luksRaw, err := runner.Output(
		ctx,
		"cryptsetup",
		"luksDump",
		"--dump-json-metadata",
		rootPart,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"read LUKS2 JSON metadata: %w",
			err,
		)
	}

	payloadOffset, err := parseLUKSPayloadOffset(luksRaw)
	if err != nil {
		return Topology{}, err
	}

	btrfsRaw, err := runner.Output(
		ctx,
		"btrfs",
		"filesystem",
		"show",
		"--raw",
		mountpoint,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"inspect Btrfs device topology: %w",
			err,
		)
	}

	btrfsUUID, deviceID, btrfsDeviceSize, btrfsUsed, devicePath, err :=
		parseBtrfsFilesystemShow(string(btrfsRaw))
	if err != nil {
		return Topology{}, err
	}

	if filepath.Clean(devicePath) != filepath.Clean(source) {
		return Topology{}, fmt.Errorf(
			"Btrfs device mismatch: current=%q expected mapping=%q",
			devicePath,
			source,
		)
	}

	exclusiveRaw, err := runner.Output(
		ctx,
		"cat",
		filepath.Join(
			"/sys/fs/btrfs",
			btrfsUUID,
			"exclusive_operation",
		),
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"read Btrfs exclusive operation: %w",
			err,
		)
	}

	exclusive := strings.TrimSpace(string(exclusiveRaw))
	if exclusive == "" {
		exclusive = "none"
	}

	swapRaw, err := runner.Output(
		ctx,
		"swapon",
		"--show=NAME",
		"--noheadings",
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"inspect active swap: %w",
			err,
		)
	}

	rootBackedSwap, err := hasRootBackedSwap(
		ctx,
		runner,
		string(swapRaw),
		rootPart,
		source,
	)
	if err != nil {
		return Topology{}, fmt.Errorf(
			"inspect active swap backing: %w",
			err,
		)
	}

	topology := Topology{
		DiskPath:           diskPath,
		DiskGUID:           diskGUID,
		DiskSizeBytes:      diskSize,
		LogicalSectorBytes: sectorSize,

		RootPartitionPath:     rootPart,
		RootPartitionNumber:   rootPartNumber,
		RootPartitionTypeGUID: rootTypeGUID,
		RootPARTUUID:          rootPARTUUID,
		RootStartBytes:        rootStart,
		RootEndBytes:          rootEnd,
		RootSizeBytes:         rootSize,

		MappingName: mappingName,
		MappingPath: source,
		LUKSUUID:    luksUUID,

		LUKSPayloadOffsetBytes: payloadOffset,

		RootFilesystem: fs,
		RootMountpoint: mountpoint,
		RootMountedRW:  rootMountedRW,

		BtrfsUUID:               btrfsUUID,
		BtrfsDeviceID:           deviceID,
		BtrfsDeviceBytes:        btrfsDeviceSize,
		BtrfsUsedBytes:          btrfsUsed,
		BtrfsExclusiveOperation: exclusive,
		SwapActive:              rootBackedSwap,
	}

	if err := ValidateTopology(topology); err != nil {
		return Topology{}, fmt.Errorf(
			"validate discovered recovery topology: %w",
			err,
		)
	}

	return topology, nil
}

func hasRootBackedSwap(
	ctx context.Context,
	runner OutputRunner,
	raw string,
	rootPartitionPath string,
	rootMappingPath string,
) (bool, error) {
	rootPartitionPath = filepath.Clean(rootPartitionPath)
	rootMappingPath = filepath.Clean(rootMappingPath)

	for _, swapPath := range strings.Fields(raw) {
		swapPath = strings.TrimSpace(swapPath)
		if swapPath == "" {
			continue
		}

		ancestryRaw, ancestryErr := runner.Output(
			ctx,
			"lsblk",
			"-s",
			"-nro",
			"PATH",
			swapPath,
		)
		if ancestryErr == nil {
			for _, path := range strings.Fields(string(ancestryRaw)) {
				path = filepath.Clean(strings.TrimSpace(path))
				if path == rootPartitionPath || path == rootMappingPath {
					return true, nil
				}
			}
			continue
		}

		source, err := outputTrim(
			ctx,
			runner,
			"findmnt",
			"-nvro",
			"SOURCE",
			"--target",
			swapPath,
		)
		if err != nil {
			return false, fmt.Errorf(
				"resolve swap source %q: lsblk: %v; findmnt: %w",
				swapPath,
				ancestryErr,
				err,
			)
		}

		source = filepath.Clean(source)
		if source == rootPartitionPath || source == rootMappingPath {
			return true, nil
		}
	}

	return false, nil
}

func parseFindmnt(raw string) (
	source string,
	fs string,
	options string,
	err error,
) {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) < 3 {
		return "", "", "", fmt.Errorf(
			"unexpected findmnt topology output %q",
			strings.TrimSpace(raw),
		)
	}

	return fields[0],
		strings.ToLower(fields[1]),
		fields[2],
		nil
}

func mountHasOption(options string, want string) bool {
	for _, option := range strings.Split(options, ",") {
		if strings.TrimSpace(option) == want {
			return true
		}
	}
	return false
}

func parseCryptStatusDevice(raw string) (string, error) {
	match := cryptStatusDevicePattern.FindStringSubmatch(raw)
	if len(match) != 2 {
		return "", errors.New(
			"cryptsetup status did not report a backing device",
		)
	}
	return strings.TrimSpace(match[1]), nil
}

func parseLUKSPayloadOffset(raw []byte) (uint64, error) {
	var metadata luksJSONMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return 0, fmt.Errorf(
			"parse LUKS2 JSON metadata: %w",
			err,
		)
	}

	if len(metadata.Segments) != 1 {
		return 0, fmt.Errorf(
			"expected exactly one LUKS2 data segment, got %d",
			len(metadata.Segments),
		)
	}

	segment, ok := metadata.Segments["0"]
	if !ok {
		return 0, errors.New(
			"LUKS2 segment 0 is missing",
		)
	}

	if segment.Type != "crypt" {
		return 0, fmt.Errorf(
			"unsupported LUKS2 segment type %q",
			segment.Type,
		)
	}

	offset, err := strconv.ParseUint(
		strings.TrimSpace(segment.Offset),
		10,
		64,
	)
	if err != nil || offset == 0 {
		return 0, fmt.Errorf(
			"invalid LUKS2 payload offset %q",
			segment.Offset,
		)
	}

	if offset%512 != 0 {
		return 0, fmt.Errorf(
			"LUKS2 payload offset %d is not 512-byte aligned",
			offset,
		)
	}

	return offset, nil
}

func parseBtrfsFilesystemShow(
	raw string,
) (
	uuid string,
	deviceID uint64,
	deviceSize uint64,
	used uint64,
	devicePath string,
	err error,
) {
	header := btrfsHeaderPattern.FindStringSubmatch(raw)
	if len(header) != 2 {
		err = errors.New(
			"Btrfs filesystem show did not report a UUID",
		)
		return
	}

	devices := btrfsDevicePattern.FindAllStringSubmatch(raw, -1)
	if len(devices) != 1 {
		err = fmt.Errorf(
			"recovery resize requires exactly one Btrfs device, got %d",
			len(devices),
		)
		return
	}

	deviceID, err = strconv.ParseUint(devices[0][1], 10, 64)
	if err != nil || deviceID == 0 {
		err = fmt.Errorf(
			"invalid Btrfs device ID %q",
			devices[0][1],
		)
		return
	}

	deviceSize, err = strconv.ParseUint(devices[0][2], 10, 64)
	if err != nil || deviceSize == 0 {
		err = fmt.Errorf(
			"invalid Btrfs device size %q",
			devices[0][2],
		)
		return
	}

	used, err = strconv.ParseUint(devices[0][3], 10, 64)
	if err != nil {
		err = fmt.Errorf(
			"invalid Btrfs used bytes %q",
			devices[0][3],
		)
		return
	}

	uuid = strings.TrimSpace(header[1])
	devicePath = strings.TrimSpace(devices[0][4])
	return
}

func outputTrim(
	ctx context.Context,
	runner OutputRunner,
	name string,
	args ...string,
) (string, error) {
	raw, err := runner.Output(ctx, name, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func outputUint(
	ctx context.Context,
	runner OutputRunner,
	name string,
	args ...string,
) (uint64, error) {
	value, err := outputTrim(ctx, runner, name, args...)
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf(
			"parse unsigned integer %q: %w",
			value,
			err,
		)
	}

	return n, nil
}

func multiply(a, b uint64) (uint64, error) {
	if a == 0 || b == 0 {
		return 0, nil
	}
	result := a * b
	if result/a != b {
		return 0, errors.New("size arithmetic overflow")
	}
	return result, nil
}
