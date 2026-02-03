package version

import (
	"testing"
)

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should be non-empty")
	}
	// Default is 0.1.0 when built without ldflags
	if len(Version) < 3 {
		t.Errorf("Version %q too short", Version)
	}
}
