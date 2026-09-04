package nixrender

import (
	"strings"
	"testing"
)

func TestStringEscapesNixInterpolation(t *testing.T) {
	got := String(`hello ${builtins.abort "no"} \\ "world"`)
	want := `"hello \${builtins.abort \"no\"} \\\\ \"world\""`
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestRenderEscapesAllUserStrings(t *testing.T) {
	s := Settings{System: "x86_64-linux", Profile: `${builtins.abort "bad"}`, Editors: []string{`a${b}`}}
	got := string(Render(s))
	if !strings.Contains(got, `profile = "\${builtins.abort \"bad\"}";`) || !strings.Contains(got, `editors = [ "a\${b}" ];`) {
		t.Fatalf("unsafe or missing escaped output:\n%s", got)
	}
}

func TestStrings(t *testing.T) {
	if got := Strings([]string{"a", "b"}); got != `[ "a" "b" ]` {
		t.Fatalf("Strings() = %q", got)
	}
}
