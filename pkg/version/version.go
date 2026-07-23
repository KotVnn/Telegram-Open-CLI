package version

import "fmt"

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func Info() string {
	return fmt.Sprintf("toc %s (commit: %s, built: %s)", Version, Commit, BuildTime)
}
