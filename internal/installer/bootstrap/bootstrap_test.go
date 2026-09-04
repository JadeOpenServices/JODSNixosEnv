package bootstrap

import (
	"strings"
	"testing"
)

func TestTransformExistingListAndOptions(t *testing.T) {
	in := []byte("{ pkgs, ... }:\n{\n  environment.systemPackages = with pkgs; [ git ];\n  environment.variables.GTK_THEME = \"old\";\n}\n")
	out := string(Transform(in, []string{"git", "fwupd"}))
	for _, want := range []string{"git  fwupd ]", "services.fwupd.enable = true;", "GTK_THEME = \"Adwaita:dark\""} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
}

func TestTransformCreatesListIdempotently(t *testing.T) {
	in := []byte("{\n}\n")
	once := Transform(in, []string{"curl"})
	twice := Transform(once, []string{"curl"})
	if string(once) != string(twice) {
		t.Fatalf("not idempotent:\n%s\n---\n%s", once, twice)
	}
}
