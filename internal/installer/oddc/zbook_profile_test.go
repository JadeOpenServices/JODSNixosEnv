package oddc

import "testing"

func TestEmbeddedRepositoryResolvesHPZBookX2G4(
	t *testing.T,
) {
	resolved, err := repositoryODDCSource(t).Resolve(
		Identity{
			FormFactor:  "laptop",
			SysVendor:   "HP",
			ProductName: "HP ZBook x2 G4",
			BoardName:   "824C",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	const want = "model/hp/zbook-x2-g4"

	if resolved.ModelID != want {
		t.Fatalf(
			"ModelID=%q want=%q",
			resolved.ModelID,
			want,
		)
	}
}

func TestEmbeddedRepositoryZBookMatchSurvivesFirmwareRevisionChanges(
	t *testing.T,
) {
	source := repositoryODDCSource(t)

	for _, revision := range []string{
		"",
		"78.21",
		"99.99-test",
	} {
		resolved, err := source.Resolve(
			Identity{
				FormFactor:     "laptop",
				SysVendor:      "HP",
				ProductName:    "HP ZBook x2 G4",
				ProductVersion: revision,
				BoardName:      "824C",
				BoardVersion:   revision,
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		if resolved.ModelID != "model/hp/zbook-x2-g4" {
			t.Fatalf(
				"revision %q resolved ModelID=%q",
				revision,
				resolved.ModelID,
			)
		}
	}
}
