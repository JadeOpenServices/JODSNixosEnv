package modelbroker

import (
	"encoding/json"
	"testing"
)

func TestValidateModelAllowsDerivedModel(t *testing.T) {
	in := []byte(
		`{"model":"gjallaros-caveman-ai","messages":[]}`,
	)

	out, err := validateModel(
		in,
		"gjallaros-caveman-ai",
	)

	if err != nil {
		t.Fatal(err)
	}

	if string(out) != string(in) {
		t.Fatalf(
			"unexpected rewrite: %s",
			out,
		)
	}
}

func TestValidateModelNormalizesProviderPrefix(t *testing.T) {
	in := []byte(
		`{"model":"ollama/gjallaros-caveman-ai","messages":[]}`,
	)

	out, err := validateModel(
		in,
		"gjallaros-caveman-ai",
	)

	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any

	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}

	if got["model"] != "gjallaros-caveman-ai" {
		t.Fatalf(
			"unexpected model: %v",
			got["model"],
		)
	}
}

func TestValidateModelRejectsBaseModel(t *testing.T) {
	_, err := validateModel(
		[]byte(
			`{"model":"qwen2.5-coder:14b"}`,
		),
		"gjallaros-caveman-ai",
	)

	if err == nil {
		t.Fatal("expected model rejection")
	}
}
