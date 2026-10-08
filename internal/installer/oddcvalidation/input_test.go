package oddcvalidation

import (
	"context"
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/discovery"
)

func TestRuntimeInputGateMatchesGeneratedState(t *testing.T) {
	runner := func(
		ctx context.Context,
		dir string,
		name string,
		args ...string,
	) ([]byte, error) {
		return []byte(`{"touchscreen":true,"penTablet":true}`), nil
	}

	detect := func(string) discovery.Hardware {
		return discovery.Hardware{
			Touchscreen: true,
			PenTablet:   true,
		}
	}

	result := runtimeInputResult(
		context.Background(),
		runner,
		detect,
		"/repo",
	)

	if !result.Passed {
		t.Fatalf("matching input state rejected: %+v", result)
	}
}

func TestRuntimeInputGateRejectsMismatch(t *testing.T) {
	runner := func(
		ctx context.Context,
		dir string,
		name string,
		args ...string,
	) ([]byte, error) {
		return []byte(`{"touchscreen":true,"penTablet":false}`), nil
	}

	detect := func(string) discovery.Hardware {
		return discovery.Hardware{
			Touchscreen: false,
			PenTablet:   false,
		}
	}

	result := runtimeInputResult(
		context.Background(),
		runner,
		detect,
		"/repo",
	)

	if result.Passed {
		t.Fatal("mismatched input state was accepted")
	}
}
