package oddccli

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"

	portable "github.com/JadeOpenServices/oddc/pkg/oddc"
)

// UpstreamAPI is the GitHub API CheckUpdate asks.
var UpstreamAPI = portable.GitHubAPI

// CheckUpdate tells whether ODDC's branch moved past the commit flake.lock
// pins. It only reports: the pin moves with `rebuild --hardware-update`.
// Offline or on any error it stays quiet, so a rebuild never waits on it.
func CheckUpdate(repo string, stdout io.Writer) {
	pinned, ref, err := oddc.LockedInput(repo)
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	upstream, err := oddc.UpstreamRevision(client, UpstreamAPI, portable.Repository, ref)
	if err != nil || upstream == pinned {
		return
	}

	fmt.Fprintf(
		stdout,
		"[GjallarOS] ODDC update available: %s -> %s. Apply with: rebuild --hardware-update\n",
		short(pinned),
		short(upstream),
	)
}
