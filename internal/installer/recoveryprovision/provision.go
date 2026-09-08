package recoveryprovision

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/bakanura/gjallarOS/internal/installer/diskplan"
	"github.com/bakanura/gjallarOS/internal/installer/gptprovision"
	"github.com/bakanura/gjallarOS/internal/installer/prompt"
	"github.com/bakanura/gjallarOS/internal/installer/recoveryresize"
)

const (
	GiB = uint64(1024 * 1024 * 1024)

	RecoveryBytes           = 12 * GiB
	GPTSafetyMarginBytes    = 1 * GiB
	FilesystemHeadroomBytes = 8 * GiB
	AlignmentBytes          = 1 * 1024 * 1024
)

type Runner interface {
	recoveryresize.OutputRunner
	gptprovision.Runner
}

type Input struct {
	Plan diskplan.Plan
	UI   prompt.UI
	Out  io.Writer

	RootMountpoint            string
	ExpectedRootPartitionPath string
	ExpectedMappingPath       string

	Unattended bool
}

type Result struct {
	Status   string
	Topology recoveryresize.Topology
	Resize   recoveryresize.Plan

	RecoveryPartition string
}

const (
	StatusReady          = "ready"
	StatusResizeRequired = "resize-required"
)

func Prepare(
	ctx context.Context,
	runner Runner,
	input Input,
) (Result, error) {
	if input.Out == nil {
		return Result{}, errors.New(
			"recovery provisioning output writer is required",
		)
	}

	if err := input.Plan.Validate(); err != nil {
		return Result{}, fmt.Errorf(
			"validate canonical disk plan: %w",
			err,
		)
	}

	if input.Plan.Recovery == nil {
		return Result{}, errors.New(
			"canonical disk plan has no recovery partition",
		)
	}

	if input.Plan.Recovery.SizeBytes != RecoveryBytes {
		return Result{}, fmt.Errorf(
			"canonical recovery partition must be exactly %d bytes, got %d",
			RecoveryBytes,
			input.Plan.Recovery.SizeBytes,
		)
	}

	// SECURITY BOUNDARY:
	//
	// Discover and validate the existing encrypted Btrfs root BEFORE calling
	// gptprovision, because gptprovision may create a partition immediately
	// when sufficient unallocated space already exists.
	//
	// Therefore non-Btrfs / unsupported root layouts fail before any GPT
	// mutation is possible.
	topology, err := recoveryresize.DiscoverTopology(
		ctx,
		runner,
		recoveryresize.DiscoveryInput{
			RootMountpoint:            input.RootMountpoint,
			ExpectedDiskPath:          input.Plan.TargetDisk.Path,
			ExpectedRootPartitionPath: input.ExpectedRootPartitionPath,
			ExpectedMappingPath:       input.ExpectedMappingPath,
		},
	)
	if err != nil {
		return Result{}, fmt.Errorf(
			"inspect existing Btrfs recovery topology: %w",
			err,
		)
	}

	if err := recoveryresize.ValidateTopology(topology); err != nil {
		return Result{}, err
	}

	// Only after the Btrfs/LUKS topology is proven may GPT provisioning inspect
	// whether the exact recovery partition already exists or can be created in
	// existing free space.
	gpt, err := gptprovision.ProvisionWithRunner(
		ctx,
		runner,
		gptprovision.Input{
			Plan:       input.Plan,
			UI:         input.UI,
			Out:        input.Out,
			Unattended: input.Unattended,
		},
	)
	if err != nil {
		return Result{}, err
	}

	switch gpt.Status {
	case gptprovision.StatusCreated,
		gptprovision.StatusAlreadyPresent:
		return Result{
			Status:            StatusReady,
			Topology:          topology,
			RecoveryPartition: gpt.PartitionPath,
		}, nil

	case gptprovision.StatusResizeRequired:
		// Continue below and build a read-only resize plan.

	default:
		return Result{}, fmt.Errorf(
			"unexpected recovery GPT provisioning state %q",
			gpt.Status,
		)
	}

	resize, err := recoveryresize.BuildPlan(
		topology,
		recoveryresize.Requirements{
			RecoveryBytes:           RecoveryBytes,
			SafetyMarginBytes:       GPTSafetyMarginBytes,
			FilesystemHeadroomBytes: FilesystemHeadroomBytes,
			AlignmentBytes:          AlignmentBytes,
		},
	)
	if err != nil {
		return Result{}, fmt.Errorf(
			"plan Btrfs recovery resize: %w",
			err,
		)
	}

	fmt.Fprintf(
		input.Out,
		"PLAN: Btrfs root requires trusted maintenance resize; root end %d -> %d; recovery=%d bytes\n",
		topology.RootEndBytes,
		resize.NewRootEndBytes,
		RecoveryBytes,
	)

	return Result{
		Status:   StatusResizeRequired,
		Topology: topology,
		Resize:   resize,
	}, nil
}
