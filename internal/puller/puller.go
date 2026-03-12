// Package puller handles HTTP file downloads with atomic writes.
package puller

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"http-pull/internal/config"
)

const defaultUserAgent = "http-pull/1.0"

// Puller downloads a file from a configured HTTP target.
type Puller struct {
	client *http.Client
	target config.TargetConfig
	logger *slog.Logger
}

// New creates a Puller for the given target configuration.
func New(target config.TargetConfig, logger *slog.Logger) *Puller {
	client := &http.Client{}

	if target.HTTPRequest != nil && target.HTTPRequest.FollowRedirects != nil && !*target.HTTPRequest.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return &Puller{
		client: client,
		target: target,
		logger: logger,
	}
}

// Pull downloads the target URL and writes it atomically to the destination.
// It returns nil on success or an error if the download or write fails.
func (p *Puller) Pull(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, p.method(), p.target.URL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Set default user agent.
	req.Header.Set("User-Agent", defaultUserAgent)

	// Apply configured headers (may override User-Agent).
	if p.target.HTTPRequest != nil {
		for _, h := range p.target.HTTPRequest.Headers {
			req.Header.Set(h.Name, h.Value)
		}
		if p.target.HTTPRequest.BasicAuth != nil {
			req.SetBasicAuth(
				p.target.HTTPRequest.BasicAuth.Username,
				p.target.HTTPRequest.BasicAuth.Password,
			)
		}
	}

	p.logger.Debug("pulling target", "url", p.target.URL)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	if err := p.atomicWrite(resp.Body); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	p.logger.Info("pull completed", "url", p.target.URL, "status", resp.StatusCode)
	return nil
}

func (p *Puller) method() string {
	if p.target.HTTPRequest != nil && p.target.HTTPRequest.Method != "" {
		return p.target.HTTPRequest.Method
	}
	return "GET"
}

func (p *Puller) atomicWrite(r io.Reader) error {
	dir := filepath.Dir(p.target.Destination)

	tmp, err := os.CreateTemp(dir, ".http-pull-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := p.applyFilePermissions(tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("applying file permissions: %w", err)
	}

	if err := os.Rename(tmpPath, p.target.Destination); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

func (p *Puller) applyFilePermissions(path string) error {
	fc := p.target.File
	if fc == nil {
		return nil
	}

	if fc.Mode != "" {
		parsed, err := strconv.ParseUint(fc.Mode, 8, 32)
		if err != nil {
			return fmt.Errorf("parsing mode %q: %w", fc.Mode, err)
		}
		if err := os.Chmod(path, os.FileMode(parsed)); err != nil {
			return fmt.Errorf("chmod: %w", err)
		}
		p.logger.Debug("applied file mode", "path", path, "mode", fc.Mode)
	}

	if fc.Owner != "" || fc.Group != "" {
		uid := -1
		gid := -1

		if fc.Owner != "" {
			id, err := resolveUID(fc.Owner)
			if err != nil {
				return fmt.Errorf("resolving owner %q: %w", fc.Owner, err)
			}
			uid = id
		}

		if fc.Group != "" {
			id, err := resolveGID(fc.Group)
			if err != nil {
				return fmt.Errorf("resolving group %q: %w", fc.Group, err)
			}
			gid = id
		}

		if err := os.Chown(path, uid, gid); err != nil {
			return fmt.Errorf("chown: %w", err)
		}
		p.logger.Debug("applied file ownership", "path", path, "uid", uid, "gid", gid)
	}

	return nil
}

func resolveUID(name string) (int, error) {
	if id, err := strconv.Atoi(name); err == nil {
		return id, nil
	}
	u, err := user.Lookup(name)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(u.Uid)
}

func resolveGID(name string) (int, error) {
	if id, err := strconv.Atoi(name); err == nil {
		return id, nil
	}
	g, err := user.LookupGroup(name)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(g.Gid)
}
