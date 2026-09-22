package usbguardsource

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
)

// Apply uses runtime actions only. USBGuard's persistent allowlist is never
// mutated; after restart every authorization is derived again from signed state.
func (s LiveSource) Apply(ctx context.Context, policy []usbtrust.Decision) error {
	output, err := s.Runner.Output(ctx, s.Binary, "list-devices")
	if err != nil {
		return err
	}
	live, err := ParseLines(string(output))
	if err != nil {
		return err
	}
	current := make(map[string]Observation, len(live))
	for _, o := range live {
		current[o.Device.RuntimeID] = o
	}
	// Revalidate the whole plan before permitting anything. Block transitions
	// precede allows, including revocation after a permanent record is forgotten.
	plan := append([]usbtrust.Decision(nil), policy...)
	sort.SliceStable(plan, func(i, j int) bool {
		return plan[i].Target == usbtrust.TargetBlock && plan[j].Target != usbtrust.TargetBlock
	})
	for _, d := range plan {
		id, err := strconv.ParseUint(d.RuntimeID, 10, 32)
		if err != nil || id == 0 || strconv.FormatUint(id, 10) != d.RuntimeID {
			return fmt.Errorf("invalid enforcement runtime ID")
		}
		if d.Target != usbtrust.TargetAllow && d.Target != usbtrust.TargetBlock {
			return fmt.Errorf("invalid enforcement target")
		}
		o, exists := current[d.RuntimeID]
		if !exists {
			return fmt.Errorf("USB device disconnected before enforcement")
		}
		if !sameIdentity(o.Device.Identity, d.Identity) {
			return fmt.Errorf("USB identity changed before enforcement")
		}
	}
	for _, d := range plan {
		if current[d.RuntimeID].Target == string(d.Target) {
			continue
		}
		if _, err := s.Runner.Output(ctx, s.Binary, string(d.Target)+"-device", d.RuntimeID); err != nil {
			return err
		}
	}
	return nil
}

func (s LiveSource) BlockAll(ctx context.Context) error {
	devices, err := s.Observed(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, d := range devices {
		// Revocation is best effort for every device. A disconnect must not
		// prevent revoking the remaining devices when verified state fails.
		if _, err := s.Runner.Output(ctx, s.Binary, "block-device", d.RuntimeID); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func sameIdentity(a, b usbtrust.Identity) bool {
	a.Interfaces = append([]string(nil), a.Interfaces...)
	b.Interfaces = append([]string(nil), b.Interfaces...)
	sort.Strings(a.Interfaces)
	sort.Strings(b.Interfaces)
	return reflect.DeepEqual(a, b)
}
