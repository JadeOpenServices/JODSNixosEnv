package workpassword

import "testing"

func TestPath(t *testing.T) {
	got, err := Path("baka-corp")
	if err != nil || got != "/var/lib/gjallarOS/passwords/baka-corp.hash" {
		t.Fatalf("%q %v", got, err)
	}
	for _, v := range []string{"", "../root", "Bad User"} {
		if _, err := Path(v); err == nil {
			t.Fatalf("accepted %q", v)
		}
	}
}
