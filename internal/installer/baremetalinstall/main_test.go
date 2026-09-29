package baremetalinstall

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	buildLimits = func() []string { return nil }
	stageFlake = func(repo string) (string, func(), error) { return repo, func() {}, nil }
	os.Exit(m.Run())
}
