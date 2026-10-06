package oddc

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	portable "github.com/JadeOpenServices/oddc"
)

// Channel is the ODDC branch the installer asks: main, its stable channel.
const Channel = "main"

// FetchedSource asks ODDC for this machine only: on first resolve it
// fetches the matched model's answer (reference closure, evidence and
// revision) into Root, replacing an earlier answer, and resolves from it.
// Nothing else of the catalog reaches the machine.
type FetchedSource struct {
	Root    string
	Channel string
	Client  *http.Client

	fetched bool
	err     error
}

func (source *FetchedSource) answer() EmbeddedSource {
	return EmbeddedSource{
		Root:       source.Root,
		Repository: "github:" + portable.Repository,
		Revision:   portable.DirSource{Root: source.Root}.Revision(),
	}
}

func (source *FetchedSource) Metadata() SourceMetadata {
	return source.answer().Metadata()
}

// fetch asks ODDC once; later calls return the first outcome.
func (source *FetchedSource) fetch(identity Identity) error {
	if !source.fetched {
		source.err = source.ask(identity)
		source.fetched = true
	}

	return source.err
}

func (source *FetchedSource) ask(identity Identity) error {
	client := source.Client
	if client == nil {
		client = &http.Client{Timeout: time.Minute}
	}
	channel := source.Channel
	if channel == "" {
		channel = Channel
	}

	upstream, err := portable.NewGitHubSource(
		client,
		portable.GitHubAPI,
		portable.GitHubRaw,
		portable.Repository,
		channel,
	)
	if err != nil {
		return fmt.Errorf("ask ODDC: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(source.Root), 0o755); err != nil {
		return err
	}
	next := source.Root + ".next"
	if err := os.RemoveAll(next); err != nil {
		return err
	}

	_, err = portable.Fetch(
		upstream,
		portable.MachineIdentity{
			FormFactor:     identity.FormFactor,
			SysVendor:      identity.SysVendor,
			ProductName:    identity.ProductName,
			ProductVersion: identity.ProductVersion,
			BoardVendor:    identity.BoardVendor,
			BoardName:      identity.BoardName,
			BoardVersion:   identity.BoardVersion,
		},
		next,
	)
	if errors.Is(err, portable.ErrNoModelMatch) {
		// No model: no answer, not an earlier machine's.
		return errors.Join(os.RemoveAll(source.Root), ErrNoMatch)
	}
	if err != nil {
		return fmt.Errorf("fetch ODDC answer: %w", err)
	}

	if err := os.RemoveAll(source.Root); err != nil {
		return err
	}
	return os.Rename(next, source.Root)
}

func (source *FetchedSource) Resolve(identity Identity) (Resolved, error) {
	if err := source.fetch(identity); err != nil {
		return Resolved{}, err
	}

	return source.answer().Resolve(identity)
}

func (source *FetchedSource) ResolveWithHost(
	identity Identity,
	host []HostOverlay,
) (Resolved, error) {
	if err := source.fetch(identity); err != nil {
		return Resolved{}, err
	}

	return source.answer().ResolveWithHost(identity, host)
}
