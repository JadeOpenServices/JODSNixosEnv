package usbtrust

import (
	"fmt"
	"os"
	"path/filepath"
)

// SignedStatePresent distinguishes a never-enrolled machine from a broken
// signed-state pair.
//
// Neither file present:
//
//	valid unenrolled state.
//
// Exactly one file present:
//
//	corruption/tampering, fail closed.
//
// Both files present:
//
//	caller must cryptographically verify them before use.
func SignedStatePresent(dir string) (bool, error) {
	trustPath := filepath.Join(
		dir,
		TrustFile,
	)
	signaturePath := filepath.Join(
		dir,
		SignatureFile,
	)

	trustPresent, err := pathPresent(trustPath)
	if err != nil {
		return false, err
	}

	signaturePresent, err := pathPresent(signaturePath)
	if err != nil {
		return false, err
	}

	switch {
	case !trustPresent && !signaturePresent:
		return false, nil

	case trustPresent && signaturePresent:
		return true, nil

	default:
		return false, fmt.Errorf(
			"USB trust signed state is incomplete: trust=%t signature=%t",
			trustPresent,
			signaturePresent,
		)
	}
}

func pathPresent(path string) (bool, error) {
	_, err := os.Lstat(path)

	switch {
	case err == nil:
		return true, nil

	case os.IsNotExist(err):
		return false, nil

	default:
		return false, fmt.Errorf(
			"inspect USB trust state %q: %w",
			path,
			err,
		)
	}
}
