package broker

import "fmt"

// AuthorizePeer is deliberately conservative.
//
// The configured desktop user may inspect status/audit state.
// Mutation requests remain root-only until GjallarOS wires an
// explicit interactive authorization mechanism.
func AuthorizePeer(
	peerUID uint32,
	action Action,
	ownerUID uint32,
) error {
	if !action.Valid() {
		return fmt.Errorf(
			"cannot authorize unknown USB trust action %q",
			action,
		)
	}

	if peerUID == 0 {
		return nil
	}

	if peerUID != ownerUID {
		return fmt.Errorf(
			"USB trust peer UID %d is not authorized",
			peerUID,
		)
	}

	if action.Mutation() {
		return fmt.Errorf(
			"USB trust mutation %q requires privileged authorization",
			action,
		)
	}

	return nil
}
