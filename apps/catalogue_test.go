package apps

import "testing"

func TestCatalogue(t *testing.T) {
	catalogue, err := Catalogue()
	if err != nil {
		t.Fatal(err)
	}
	asked := map[string]bool{}
	for _, app := range catalogue {
		if app.Name == "" || app.Category == "" {
			t.Errorf("%s: name and category are required", app.ID)
		}
		if app.Installer != "" {
			asked[app.ID] = true
		}
	}
	for _, id := range []string{"ai", "containers", "nemu", "tailscale"} {
		if !asked[id] {
			t.Errorf("%s: installer question missing", id)
		}
	}
	if err := Validate([]string{"ai", "git"}); err != nil {
		t.Fatal(err)
	}
	if Validate([]string{"gaming"}) == nil || Validate([]string{"git", "git"}) == nil {
		t.Fatal("bad app list accepted")
	}
}
