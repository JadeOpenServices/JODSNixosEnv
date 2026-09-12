package oddcvalidation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type Authority struct {
	Allowed  bool
	Identity string
	Repo     string
	Reason   string
}

type AuthorityRunner func(
	context.Context,
	string,
	string,
	...string,
) ([]byte, error)

func CheckContributorAuthority(
	ctx context.Context,
	repo string,
	expectedRepo string,
) Authority {
	return checkContributorAuthority(
		ctx,
		repo,
		expectedRepo,
		defaultCommandRunner,
	)
}

func checkContributorAuthority(
	ctx context.Context,
	repo string,
	expectedRepo string,
	run AuthorityRunner,
) Authority {
	remote, err := run(ctx, repo, "git", "remote", "get-url", "origin")
	if err != nil {
		return denied("cannot resolve origin")
	}

	actualRepo, err := normalizeGitHubRepo(string(remote))
	if err != nil || actualRepo != expectedRepo {
		return denied("origin is not the expected upstream repository")
	}

	status, err := run(
		ctx,
		repo,
		"git", "status", "--porcelain", "--untracked-files=all",
	)
	if err != nil {
		return denied("cannot inspect checkout state")
	}
	if strings.TrimSpace(string(status)) != "" {
		return denied("checkout contains unrelated changes")
	}

	branch, err := run(
		ctx,
		repo,
		"git", "symbolic-ref", "--quiet", "--short", "HEAD",
	)
	if err != nil || strings.TrimSpace(string(branch)) == "" {
		return denied("checkout is detached or branch state is unavailable")
	}

	userJSON, err := run(ctx, repo, "gh", "api", "user")
	if err != nil {
		return denied("GitHub identity could not be authenticated")
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(userJSON, &user); err != nil ||
		strings.TrimSpace(user.Login) == "" {
		return denied("GitHub identity response is invalid")
	}

	repoJSON, err := run(
		ctx,
		repo,
		"gh", "api", "repos/"+expectedRepo,
	)
	if err != nil {
		return denied("GitHub repository permission could not be verified")
	}

	var remoteRepo struct {
		FullName    string `json:"full_name"`
		Permissions struct {
			Push bool `json:"push"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(repoJSON, &remoteRepo); err != nil {
		return denied("GitHub repository response is invalid")
	}

	if remoteRepo.FullName != expectedRepo || !remoteRepo.Permissions.Push {
		return denied("authenticated GitHub identity lacks upstream write permission")
	}

	return Authority{
		Allowed:  true,
		Identity: user.Login,
		Repo:     actualRepo,
	}
}

func denied(reason string) Authority {
	return Authority{Reason: reason}
}

func normalizeGitHubRepo(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".git")

	if strings.HasPrefix(raw, "git@github.com:") {
		repo := strings.TrimPrefix(raw, "git@github.com:")
		if strings.Count(repo, "/") == 1 {
			return repo, nil
		}
	}

	u, err := url.Parse(raw)
	if err == nil && u.Host == "github.com" {
		repo := strings.TrimPrefix(u.Path, "/")
		if strings.Count(repo, "/") == 1 {
			return repo, nil
		}
	}

	return "", fmt.Errorf("unsupported GitHub repository URL %q", raw)
}
