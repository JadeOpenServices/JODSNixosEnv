package oddc

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// UpstreamRevision asks the GitHub API at api which commit ref names in
// ODDC's repository. "" asks for the default branch.
func UpstreamRevision(client *http.Client, api, repository, ref string) (string, error) {
	if ref == "" {
		ref = "HEAD"
	}
	request, err := http.NewRequest(
		http.MethodGet,
		strings.TrimSuffix(api, "/")+"/repos/"+repository+"/commits/"+url.PathEscape(ref),
		nil,
	)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github.sha")

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 128))
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ask ODDC upstream: %s", response.Status)
	}

	rev := strings.TrimSpace(string(body))
	if len(rev) != 40 || strings.Trim(rev, "0123456789abcdef") != "" {
		return "", errors.New("ask ODDC upstream: answer is not a commit")
	}
	return rev, nil
}
