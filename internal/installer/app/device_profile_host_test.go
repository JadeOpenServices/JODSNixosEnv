package app

import (
	"context"
	"errors"
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/discovery"
	"github.com/JadeOpenServices/gjallarOS/internal/installer/oddc"
	"github.com/JadeOpenServices/gjallarOS/internal/installer/oddchost"
	portable "github.com/JadeOpenServices/oddc/pkg/oddc"
)

type hostReadTestSource struct {
	hostCalls int
}

func (source *hostReadTestSource) Resolve(
	oddc.Identity,
) (oddc.Resolved, error) {
	return oddc.Resolved{
		ModelID: "model/test/laptop",
		Source:  source.Metadata(),
	}, nil
}

func (source *hostReadTestSource) ResolveWithHost(
	_ oddc.Identity,
	host []oddc.HostOverlay,
) (oddc.Resolved, error) {
	source.hostCalls++

	if len(host) != 1 {
		return oddc.Resolved{}, errors.New(
			"expected exactly one host overlay",
		)
	}

	return oddc.Resolved{
		ModelID: "model/test/laptop",
		Source:  source.Metadata(),
	}, nil
}

func (*hostReadTestSource) Metadata() oddc.SourceMetadata {
	return oddc.SourceMetadata{
		Kind:       "embedded",
		Repository: "embedded:test",
		Revision:   "test-revision",
	}
}

func TestResolveODDCModelWithHostUsesMountedInstalledRoot(t *testing.T) {
	previous := loadODDCHostOverlay
	defer func() {
		loadODDCHostOverlay = previous
	}()

	var gotPath string
	var gotModel string

	loadODDCHostOverlay = func(
		_ context.Context,
		path string,
		modelID string,
	) (oddc.HostOverlay, bool, error) {
		gotPath = path
		gotModel = modelID

		return oddc.HostOverlay{
			APIVersion:  "oddc.openjade.de/v2",
			ID:          "host/machine-local",
			Kind:        "host",
			TargetModel: modelID,
			Overrides: map[string]any{
				"hardware": map[string]any{},
			},
		}, true, nil
	}

	source := &hostReadTestSource{}
	base := oddc.Resolved{
		ModelID: "model/test/laptop",
		Source:  source.Metadata(),
	}

	resolved, host, err := resolveODDCModelWithHost(
		context.Background(),
		"/mnt",
		source,
		discovery.Hardware{
			FormFactor:  "laptop",
			SysVendor:   "Test",
			ProductName: "Laptop",
		},
		base,
	)
	if err != nil {
		t.Fatal(err)
	}

	wantPath := "/mnt/var/lib/gjallarOS/oddc/host-overlay.json"
	if gotPath != wantPath {
		t.Fatalf(
			"host overlay path = %q, want %q",
			gotPath,
			wantPath,
		)
	}
	if gotModel != base.ModelID {
		t.Fatalf(
			"host overlay model = %q, want %q",
			gotModel,
			base.ModelID,
		)
	}
	if host == nil {
		t.Fatal("loaded host overlay was not returned")
	}
	if source.hostCalls != 1 {
		t.Fatalf(
			"host-aware resolution calls = %d, want 1",
			source.hostCalls,
		)
	}
	if resolved.ModelID != base.ModelID {
		t.Fatalf(
			"resolved model = %q, want %q",
			resolved.ModelID,
			base.ModelID,
		)
	}
}

func TestResolveODDCModelWithHostMissingPreservesBase(t *testing.T) {
	previous := loadODDCHostOverlay
	defer func() {
		loadODDCHostOverlay = previous
	}()

	loadODDCHostOverlay = func(
		context.Context,
		string,
		string,
	) (oddc.HostOverlay, bool, error) {
		return oddc.HostOverlay{}, false, nil
	}

	source := &hostReadTestSource{}
	base := oddc.Resolved{
		ModelID: "model/test/laptop",
		Source:  source.Metadata(),
	}

	resolved, host, err := resolveODDCModelWithHost(
		context.Background(),
		"/",
		source,
		discovery.Hardware{},
		base,
	)
	if err != nil {
		t.Fatal(err)
	}

	if host != nil {
		t.Fatal("missing persisted overlay returned host state")
	}
	if source.hostCalls != 0 {
		t.Fatalf(
			"host-aware source called with no persisted overlay: %d",
			source.hostCalls,
		)
	}
	if resolved.ModelID != base.ModelID {
		t.Fatalf(
			"base resolution changed: %+v",
			resolved,
		)
	}
}

func TestResolveODDCModelWithHostUnmatchedSkipsStateRead(t *testing.T) {
	previous := loadODDCHostOverlay
	defer func() {
		loadODDCHostOverlay = previous
	}()

	called := false
	loadODDCHostOverlay = func(
		context.Context,
		string,
		string,
	) (oddc.HostOverlay, bool, error) {
		called = true
		return oddc.HostOverlay{}, false, nil
	}

	resolved, host, err := resolveODDCModelWithHost(
		context.Background(),
		"/",
		&hostReadTestSource{},
		discovery.Hardware{},
		oddc.Resolved{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("host state read attempted for unmatched ODDC model")
	}
	if host != nil {
		t.Fatal("unmatched ODDC model returned host state")
	}
	if resolved.ModelID != "" {
		t.Fatalf("unexpected model %q", resolved.ModelID)
	}
}

func TestShouldApplyODDCHostOverlay(t *testing.T) {
	tests := []struct {
		name       string
		persistent bool
		rebind     bool
		want       bool
	}{
		{
			name:       "existing bound host",
			persistent: true,
			want:       true,
		},
		{
			name:       "fresh install",
			persistent: false,
			want:       false,
		},
		{
			name:       "device rebind",
			persistent: true,
			rebind:     true,
			want:       false,
		},
	}

	for _, test := range tests {
		got := shouldApplyODDCHostOverlay(
			test.persistent,
			test.rebind,
		)
		if got != test.want {
			t.Fatalf(
				"%s: got %v, want %v",
				test.name,
				got,
				test.want,
			)
		}
	}
}

func TestHostOverlayForCommitUnchangedSameMachineDoesNothing(
	t *testing.T,
) {
	got, err := hostOverlayForCommit(
		nil,
		false,
		false,
		"model/test/laptop",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("unchanged same-machine state unexpectedly scheduled a commit")
	}
}

func TestHostOverlayForCommitUsesChangedState(t *testing.T) {
	host, err := oddchost.New("model/test/laptop")
	if err != nil {
		t.Fatal(err)
	}

	if err := oddchost.AddDeletion(
		&host,
		"hardware.security.fingerprint.primary",
	); err != nil {
		t.Fatal(err)
	}

	got, err := hostOverlayForCommit(
		&host,
		true,
		false,
		"model/test/laptop",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("changed host state was not scheduled for commit")
	}

	value, exists := portable.Lookup(
		got.Overrides,
		"hardware.security.fingerprint.primary.$delete",
	)
	if !exists || value != true {
		t.Fatalf(
			"staged deletion lost before commit: %#v",
			got.Overrides,
		)
	}
}

func TestHostOverlayForCommitRebindCreatesFreshModelState(
	t *testing.T,
) {
	got, err := hostOverlayForCommit(
		nil,
		false,
		true,
		"model/new/laptop",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("rebind did not create replacement host state")
	}
	if got.TargetModel != "model/new/laptop" {
		t.Fatalf(
			"rebind host target = %q",
			got.TargetModel,
		)
	}
	if len(got.Overrides) != 0 {
		t.Fatalf(
			"rebind inherited stale host overrides: %#v",
			got.Overrides,
		)
	}
}

func TestHostOverlayForCommitRejectsWrongModel(t *testing.T) {
	host, err := oddchost.New("model/old/laptop")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := hostOverlayForCommit(
		&host,
		true,
		false,
		"model/new/laptop",
	); err == nil {
		t.Fatal("wrong-model staged host state was accepted")
	}
}

func TestSaveODDCHostOverlayUsesInstalledRoot(t *testing.T) {
	previous := saveODDCHostOverlayPrivileged
	defer func() {
		saveODDCHostOverlayPrivileged = previous
	}()

	host, err := oddchost.New("model/test/laptop")
	if err != nil {
		t.Fatal(err)
	}

	var gotPath string

	saveODDCHostOverlayPrivileged = func(
		_ context.Context,
		path string,
		overlay portable.Overlay,
	) error {
		gotPath = path

		if overlay.TargetModel != "model/test/laptop" {
			t.Fatalf(
				"saved target model = %q",
				overlay.TargetModel,
			)
		}

		return nil
	}

	if err := saveODDCHostOverlay(
		context.Background(),
		"/mnt",
		&host,
	); err != nil {
		t.Fatal(err)
	}

	want := "/mnt/var/lib/gjallarOS/oddc/host-overlay.json"
	if gotPath != want {
		t.Fatalf(
			"saved path = %q, want %q",
			gotPath,
			want,
		)
	}
}
