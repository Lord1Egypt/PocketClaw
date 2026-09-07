package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// readAgentSourceFile returns the contents of one file in this package, for
// the small number of guards that must assert on source rather than on
// behaviour because the code path they protect is not reachable at runtime.
func readAgentSourceFile(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(".", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(raw)
}
