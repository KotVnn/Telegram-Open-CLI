package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

const (
	// GitHubAPIURL is the GitHub API URL for releases.
	GitHubAPIURL = "https://api.github.com/repos/KotVnn/Telegram-Open-CLI/releases/latest"
	// CheckInterval is how often to check for updates.
	CheckInterval = 24 * time.Hour
)

// Release represents a GitHub release.
type Release struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

// Checker checks for application updates.
type Checker struct {
	currentVersion string
	logger         zerolog.Logger
	client         *http.Client
}

// NewChecker creates a new update checker.
func NewChecker(currentVersion string, logger zerolog.Logger) *Checker {
	return &Checker{
		currentVersion: currentVersion,
		logger:         logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CheckForUpdate checks if a newer version is available.
func (c *Checker) CheckForUpdate(ctx context.Context) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, GitHubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("check update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("unmarshal release: %w", err)
	}

	return &release, nil
}

// IsNewerVersion checks if the remote version is newer than the current version.
func (c *Checker) IsNewerVersion(remoteVersion string) bool {
	// Simple string comparison - not semver aware
	// For production, consider using a proper semver library
	return remoteVersion != c.currentVersion
}

// Start performs an immediate check then runs periodic checks.
func (c *Checker) Start(ctx context.Context) {
	// Immediate check on startup
	c.check(ctx)

	go func() {
		ticker := time.NewTicker(CheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.check(ctx)
			}
		}
	}()
}

func (c *Checker) check(ctx context.Context) {
	release, err := c.CheckForUpdate(ctx)
	if err != nil {
		c.logger.Debug().Err(err).Msg("failed to check for update")
		return
	}

	if c.IsNewerVersion(release.TagName) {
		c.logger.Info().
			Str("current", c.currentVersion).
			Str("latest", release.TagName).
			Str("url", release.HTMLURL).
			Msg("new version available")
	}
}
