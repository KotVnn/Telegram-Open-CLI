package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultVariables(t *testing.T) {
	tests := []struct {
		name     string
		variable string
		expected string
	}{
		{"Version default", Version, "dev"},
		{"Commit default", Commit, "unknown"},
		{"BuildTime default", BuildTime, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.variable)
		})
	}
}

func TestInfo(t *testing.T) {
	origVersion := Version
	origCommit := Commit
	origBuildTime := BuildTime
	defer func() {
		Version = origVersion
		Commit = origCommit
		BuildTime = origBuildTime
	}()

	tests := []struct {
		name      string
		version   string
		commit    string
		buildTime string
		expected  string
	}{
		{
			name:      "default values",
			version:   "dev",
			commit:    "unknown",
			buildTime: "unknown",
			expected:  "toc dev (commit: unknown, built: unknown)",
		},
		{
			name:      "release values",
			version:   "1.0.0",
			commit:    "abc1234",
			buildTime: "2026-01-15",
			expected:  "toc 1.0.0 (commit: abc1234, built: 2026-01-15)",
		},
		{
			name:      "empty values",
			version:   "",
			commit:    "",
			buildTime: "",
			expected:  "toc  (commit: , built: )",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			Commit = tt.commit
			BuildTime = tt.buildTime

			result := Info()
			assert.Equal(t, tt.expected, result)
		})
	}
}
