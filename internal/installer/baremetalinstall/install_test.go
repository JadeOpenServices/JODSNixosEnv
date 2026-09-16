package baremetalinstall

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls   [][]string
	outputs map[string][]byte
	errors  map[string]error
	onRun   func()
}

func commandKey(name string, args ...string) string {
	return strings.Join(append([]string{name}, args...), "\x00")
}

func (f *fakeRunner) Run(
	_ context.Context,
	_ io.Reader,
	_ io.Writer,
	_ io.Writer,
	name string,
	args ...string,
) error {
	call := append([]string{name}, args...)
	f.calls = append(f.calls, call)

	if f.onRun != nil {
		f.onRun()
	}

	return f.errors[commandKey(name, args...)]
}

func (f *fakeRunner) Output(
	_ context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	call := append([]string{name}, args...)
	f.calls = append(f.calls, call)

	k := commandKey(name, args...)
	if err := f.errors[k]; err != nil {
		return nil, err
	}
	if output, ok := f.outputs[k]; ok {
		return output, nil
	}

	return nil, fmt.Errorf("unexpected output command: %v", call)
}

func testRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(root, "flake.nix"),
		[]byte("{ outputs = _: {}; }\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(root, "generated"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "generated", "state.nix"),
		[]byte("{ hostname = \"gjallarOS\"; }\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(root, "user.config.json"),
		[]byte("{\"profile\":\"laptop\"}\n"),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	return root
}

func mountedRunner() *fakeRunner {
	return &fakeRunner{
		outputs: map[string][]byte{
			commandKey(
				"findmnt",
				"-nro",
				"TARGET",
				"--mountpoint",
				"/mnt",
			): []byte("/mnt\n"),
			commandKey(
				"findmnt",
				"-nro",
				"TARGET",
				"--mountpoint",
				"/mnt/boot",
			): []byte("/mnt/boot\n"),
		},
		errors: map[string]error{},
	}
}

func createInstalledProfile(t *testing.T) {
	t.Helper()

	path := "/mnt/nix/var/nix/profiles"
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Skipf("cannot create temporary /mnt test fixture: %v", err)
	}

	system := filepath.Join(path, "system")
	_ = os.Remove(system)

	if err := os.Symlink("/nix/store/test-system", system); err != nil {
		t.Skipf("cannot create /mnt system profile fixture: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Remove(system)
	})
}

func TestInstallUsesOnlyPreparedTarget(t *testing.T) {
	createInstalledProfile(t)

	repo := testRepo(t)
	r := mountedRunner()
	var out bytes.Buffer

	result, err := install(
		context.Background(),
		Input{
			Repo:     repo,
			Hostname: "gjallarOS",
			Out:      &out,
		},
		r,
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.Stage != "os-installed" {
		t.Fatalf("stage = %q", result.Stage)
	}

	joined := callsText(r.calls)
	required := strings.Join([]string{
		"sudo",
		"nixos-install",
		"--root",
		"/mnt",
		"--flake",
		repo + "#gjallarOS",
		"--no-root-passwd",
	}, " ")

	if !strings.Contains(joined, required) {
		t.Fatalf(
			"missing expected nixos-install invocation:\n%s",
			joined,
		)
	}

	for _, forbidden := range []string{
		"sgdisk",
		"wipefs",
		"parted",
		"resize2fs",
		"cryptsetup",
		"mkfs",
		"mount ",
		"umount",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf(
				"install stage attempted disk/filesystem operation %q:\n%s",
				forbidden,
				joined,
			)
		}
	}
}

func TestInstallRequiresRootAndBootMounts(t *testing.T) {
	repo := testRepo(t)

	tests := []struct {
		name   string
		remove string
	}{
		{name: "root", remove: "/mnt"},
		{name: "boot", remove: "/mnt/boot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := mountedRunner()
			delete(
				r.outputs,
				commandKey(
					"findmnt",
					"-nro",
					"TARGET",
					"--mountpoint",
					tt.remove,
				),
			)
			r.errors[commandKey(
				"findmnt",
				"-nro",
				"TARGET",
				"--mountpoint",
				tt.remove,
			)] = fmt.Errorf("not mounted")

			var out bytes.Buffer
			_, err := install(
				context.Background(),
				Input{
					Repo:     repo,
					Hostname: "gjallarOS",
					Out:      &out,
				},
				r,
			)

			if err == nil || !strings.Contains(err.Error(), "not mounted") {
				t.Fatalf("unmounted target accepted: %v", err)
			}
			if strings.Contains(callsText(r.calls), "nixos-install") {
				t.Fatalf(
					"nixos-install ran before mount validation:\n%s",
					callsText(r.calls),
				)
			}
		})
	}
}

func TestInstallerConfigRemainsUnchanged(t *testing.T) {
	createInstalledProfile(t)

	repo := testRepo(t)

	settingsBefore, err := os.ReadFile(
		filepath.Join(repo, "generated", "state.nix"),
	)
	if err != nil {
		t.Fatal(err)
	}
	userBefore, err := os.ReadFile(
		filepath.Join(repo, "user.config.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	r := mountedRunner()
	var out bytes.Buffer

	if _, err := install(
		context.Background(),
		Input{
			Repo:     repo,
			Hostname: "gjallarOS",
			Out:      &out,
		},
		r,
	); err != nil {
		t.Fatal(err)
	}

	settingsAfter, err := os.ReadFile(
		filepath.Join(repo, "generated", "state.nix"),
	)
	if err != nil {
		t.Fatal(err)
	}
	userAfter, err := os.ReadFile(
		filepath.Join(repo, "user.config.json"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(settingsBefore, settingsAfter) {
		t.Fatal("settings.nix changed during installation")
	}
	if !bytes.Equal(userBefore, userAfter) {
		t.Fatal("user.config.json changed during installation")
	}
}

func TestConfigMutationDuringInstallFailsClosed(t *testing.T) {
	createInstalledProfile(t)

	repo := testRepo(t)
	r := mountedRunner()

	r.onRun = func() {
		if err := os.MkdirAll(filepath.Join(repo, "generated"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(repo, "generated", "state.nix"),
			[]byte("MUTATED\n"),
			0644,
		); err != nil {
			t.Fatal(err)
		}
	}

	var out bytes.Buffer
	_, err := install(
		context.Background(),
		Input{
			Repo:     repo,
			Hostname: "gjallarOS",
			Out:      &out,
		},
		r,
	)

	if err == nil ||
		!strings.Contains(err.Error(), "installer config changed") {
		t.Fatalf("config mutation was not detected: %v", err)
	}
}

func TestInstallHasNoJODSEnrollmentOrPhoneHome(t *testing.T) {
	createInstalledProfile(t)

	repo := testRepo(t)
	r := mountedRunner()
	var out bytes.Buffer

	if _, err := install(
		context.Background(),
		Input{
			Repo:     repo,
			Hostname: "gjallarOS",
			Out:      &out,
		},
		r,
	); err != nil {
		t.Fatal(err)
	}

	joined := strings.ToLower(callsText(r.calls))

	for _, forbidden := range []string{
		"jods-mdm-agent",
		"jods-endpoint",
		"enroll.service",
		"enroll.timer",
		"installation-complete",
		"curl",
		"wget",
		"systemctl",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf(
				"install stage performed forbidden management action %q:\n%s",
				forbidden,
				joined,
			)
		}
	}
}

func TestFailureDoesNotReportOSInstalled(t *testing.T) {
	repo := testRepo(t)
	r := mountedRunner()

	r.errors[commandKey(
		"sudo",
		"nixos-install",
		"--root",
		"/mnt",
		"--flake",
		repo+"#gjallarOS",
		"--no-root-passwd",
	)] = fmt.Errorf("install failed")

	var out bytes.Buffer
	_, err := install(
		context.Background(),
		Input{
			Repo:     repo,
			Hostname: "gjallarOS",
			Out:      &out,
		},
		r,
	)

	if err == nil {
		t.Fatal("failed nixos-install reported success")
	}
	if strings.Contains(out.String(), "STAGE: os-installed") {
		t.Fatalf(
			"failed installation reported os-installed:\n%s",
			out.String(),
		)
	}
}

func callsText(calls [][]string) string {
	var lines []string
	for _, call := range calls {
		lines = append(lines, strings.Join(call, " "))
	}
	return strings.Join(lines, "\n")
}
