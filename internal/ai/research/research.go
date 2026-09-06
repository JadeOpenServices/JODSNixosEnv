// Package research implements a read-only, provenance-preserving HTTPS broker.
package research

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Result struct {
	Title     string    `json:"title"`
	Source    string    `json:"source"`
	URL       string    `json:"url"`
	Published string    `json:"publishedOrUpdated,omitempty"`
	Retrieved time.Time `json:"retrieved"`
	Content   string    `json:"content"`
}
type Broker struct {
	AllowedHosts []string
	MaxBytes     int64
	Timeout      time.Duration
	Resolver     *net.Resolver
}

func (b Broker) Fetch(ctx context.Context, raw string) (Result, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Hostname() == "" {
		return Result{}, fmt.Errorf("only credential-free HTTPS URLs are allowed")
	}
	if !b.allowed(u.Hostname()) {
		return Result{}, fmt.Errorf("host is not in research allowlist: %s", u.Hostname())
	}
	if err := b.publicHost(ctx, u.Hostname()); err != nil {
		return Result{}, err
	}
	if b.MaxBytes <= 0 {
		b.MaxBytes = 2 << 20
	}
	if b.Timeout <= 0 {
		b.Timeout = 15 * time.Second
	}
	client := &http.Client{Timeout: b.Timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 || req.URL.Scheme != "https" || !b.allowed(req.URL.Hostname()) {
			return fmt.Errorf("redirect rejected")
		}
		return b.publicHost(req.Context(), req.URL.Hostname())
	}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	req.Header.Set("User-Agent", "GjallarOS-Research/1.0")
	req.Header.Set("Accept", "text/html,text/plain,application/json")
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("upstream returned %s", resp.Status)
	}
	limited := io.LimitReader(resp.Body, b.MaxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return Result{}, err
	}
	if int64(len(body)) > b.MaxBytes {
		return Result{}, fmt.Errorf("response exceeds %d bytes", b.MaxBytes)
	}
	text := strings.TrimSpace(string(body))
	title := resp.Header.Get("X-Page-Title")
	return Result{Title: title, Source: resp.Request.URL.Hostname(), URL: resp.Request.URL.String(), Published: resp.Header.Get("Last-Modified"), Retrieved: time.Now().UTC(), Content: text}, nil
}

func (b Broker) allowed(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, a := range b.AllowedHosts {
		a = strings.ToLower(a)
		if host == a || strings.HasSuffix(host, "."+a) {
			return true
		}
	}
	return false
}
func (b Broker) publicHost(ctx context.Context, host string) error {
	resolver := b.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	ips, err := resolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("resolve research host: %w", err)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsMulticast() {
			return fmt.Errorf("research host resolved to a non-public address")
		}
	}
	return nil
}

func Handler(b Broker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		result, err := b.Fetch(r.Context(), r.URL.Query().Get("url"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	})
}
