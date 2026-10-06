package deviceprofile

import (
	"path/filepath"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

// AnswerDir is where a GjallarOS tree keeps the ODDC answer for its machine;
// the system configuration reads it as oddc.catalog.
const AnswerDir = "generated/oddc"

// CurrentSource asks ODDC for this machine's model and keeps the answer in
// repo's generated/oddc.
func CurrentSource(repo string) *oddc.FetchedSource {
	return &oddc.FetchedSource{
		Root: filepath.Join(repo, filepath.FromSlash(AnswerDir)),
	}
}
