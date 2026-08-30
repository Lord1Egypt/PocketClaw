package pcruntime

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// helperDirMu serialises helper-directory construction per tool so two
// concurrent executions cannot see a half-built directory.
var helperDirMu sync.Map // tool_id -> *sync.Mutex

// materialiseHelpers builds the directory a tool's helper executables are found
// through, and returns its path.
//
// The helpers themselves stay in nativeLibraryDir; this directory holds only
// symlinks to them. That distinction is the whole design: a tool like git looks
// its transport helper up by a logical name the APK cannot use as a filename,
// and Android will not execute a real file written into app storage. A symlink
// is not an executable — the kernel resolves it and executes the read-only
// target — so the packaged payload remains the only thing that ever runs.
func (m *Manager) materialiseHelpers(resolved *ResolvedTool) (string, error) {
	toolID := resolved.Tool.ToolID
	lock, _ := helperDirMu.LoadOrStore(toolID, &sync.Mutex{})
	mutex := lock.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	dir := filepath.Join(m.registry.paths.MetadataDir, "exec", toolID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("cannot create helper directory for %s: %w", toolID, err)
	}

	prepared := make([]string, 0, len(resolved.HelperPaths))
	for logicalName, target := range resolved.HelperPaths {
		link := filepath.Join(dir, logicalName)

		// An existing link is reused only when it already points at the payload
		// this resolution verified. After an app update the payload path can
		// change, and a stale link would silently point at nothing.
		if current, err := os.Readlink(link); err == nil && current == target {
			prepared = append(prepared, logicalName)
			continue
		}
		if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
			emitError(EventHelpersFailed, map[string]any{
				"tool": toolID, "helper": logicalName, "diagnostics": err.Error(),
			})
			return "", fmt.Errorf("cannot replace stale helper link %s: %w", link, err)
		}
		if err := os.Symlink(target, link); err != nil {
			emitError(EventHelpersFailed, map[string]any{
				"tool": toolID, "helper": logicalName, "diagnostics": err.Error(),
			})
			return "", fmt.Errorf("cannot link helper %s for %s: %w", logicalName, toolID, err)
		}
		prepared = append(prepared, logicalName)
	}

	sort.Strings(prepared)
	emitDebug(EventHelpersPrepared, map[string]any{
		"tool":      toolID,
		"exec_path": dir,
		"helpers":   prepared,
		"count":     len(prepared),
	})
	return dir, nil
}

// basicAuthValue encodes a token as HTTP Basic credentials for GitHub, which
// accepts any username alongside a token.
func basicAuthValue(token string) string {
	return base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
}
