package deviceprofile

import (
	"path/filepath"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func CurrentEmbeddedSource(
	repo string,
	revision string,
) oddc.EmbeddedSource {
	return oddc.EmbeddedSource{
		Root:       filepath.Join(repo, "oddc"),
		Repository: "embedded:oddc",
		Revision:   revision,
	}
}
