// Package version exposes the application version, injected at build time.
package version

import "runtime/debug"

// Version is the application version. It defaults to "dev" for local/unstamped
// builds and is overridden at build time via the linker:
//
//	go build -ldflags "-X github.com/aloks98/waygates/backend/internal/version.Version=$VERSION"
//
// See the Makefile (local builds) and Dockerfile/release workflow (image builds).
var Version = "dev"

// resolved is computed once at init: the stamped Version, or "dev-<commit>" when
// the version was not injected at build time (see String).
var resolved = resolve()

// String returns the application version for display. When the version was not
// stamped via -ldflags (still the default "dev"), it appends the VCS commit
// recorded in the build info, e.g. "dev-c7ee777", with a "-dirty" suffix for
// builds from an uncommitted tree. Falls back to plain "dev" when no VCS info is
// available (e.g. an image built without the .git context and no VERSION arg).
func String() string {
	return resolved
}

func resolve() string {
	if Version != "dev" {
		return Version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return Version
	}

	var revision string
	var modified bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	if revision == "" {
		return Version
	}
	if len(revision) > 7 {
		revision = revision[:7]
	}

	v := Version + "-" + revision
	if modified {
		v += "-dirty"
	}
	return v
}
