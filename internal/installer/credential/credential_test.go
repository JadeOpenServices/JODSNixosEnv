package credential

import "testing"

func TestPath(t *testing.T) {
	got, err := Path("testuser-corp")
	if err != nil || got != "/var/lib/gjallarOS/passwords/testuser-corp.hash" {
		t.Fatalf("%q %v", got, err)
	}
	for _, v := range []string{"", "../root", "Bad User"} {
		if _, err := Path(v); err == nil {
			t.Fatalf("accepted %q", v)
		}
	}
}
