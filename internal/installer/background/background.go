package background

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const maxDownload = 100 << 20

func Resolve(ctx context.Context, repo, dotfiles, role, value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value), nil
	}
	u, err := url.Parse(value)
	if err == nil && u.Scheme != "" {
		if u.Scheme != "https" || u.Host == "" {
			return "", fmt.Errorf("background URL must use HTTPS")
		}
		if role != "normal" && role != "work" && role != "gaming" {
			return "", fmt.Errorf("invalid background role: %q", role)
		}
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(u.Path), "."))
		switch ext {
		case "png", "jpg", "jpeg", "webp", "avif", "svg":
		default:
			ext = "png"
		}
		sum := sha256.Sum256([]byte(value))
		target := filepath.Join(repo, "non-nix", "wallpapers", fmt.Sprintf("user-%s-%x.%s", role, sum[:8], ext))
		if info, err := os.Stat(target); err == nil && info.Size() > 0 {
			return target, nil
		}
		if err := download(ctx, target, value); err != nil {
			return "", err
		}
		return target, nil
	}
	if !filepath.IsAbs(dotfiles) {
		return "", fmt.Errorf("dotfiles directory must be absolute")
	}
	return filepath.Clean(filepath.Join(dotfiles, value)), nil
}

func download(ctx context.Context, target, source string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download background: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("download background: HTTP %s", resp.Status)
	}
	if resp.ContentLength > maxDownload {
		return fmt.Errorf("background exceeds %d bytes", maxDownload)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".wallpaper-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	n, err := io.Copy(tmp, io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		tmp.Close()
		return err
	}
	if n > maxDownload {
		tmp.Close()
		return fmt.Errorf("background exceeds %d bytes", maxDownload)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, target)
}
