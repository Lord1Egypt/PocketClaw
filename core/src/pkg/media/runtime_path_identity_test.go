package media_test

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/media"
	"github.com/sipeed/picoclaw/pkg/utils"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Zero-Pico, runtime edition: nothing PocketClaw creates on disk may be named
// after Pico.
//
// The lexical guard could not catch this. `tool/no_active_pico.py` treats
// `core/src` as vendored upstream except for named paths, and the media cache
// constant was not one of them — so `picoclaw_media` sat in the one place that
// mattered most, the directory every downloaded attachment is written into, and
// a physical log showed new files appearing under
// `.../cache/tmp/picoclaw_media/` with every gate green.
//
// So this test is behavioural rather than lexical: it drives the real download
// helper the Telegram, Matrix, Feishu and WeCom channels all use, then walks
// what actually appeared on disk. A rename that misses a call site fails here
// even if the constant is clean.

// walkForPicoPaths returns every path under root whose name mentions Pico.
func walkForPicoPaths(t *testing.T, root string) []string {
	t.Helper()
	var offenders []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// A vanished temp file is not a finding.
			return nil
		}
		lowered := strings.ToLower(info.Name())
		if strings.Contains(lowered, "pico") {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return offenders
}

// TestDownloadedMediaCreatesNoPicoPath is the regression the owner asked for: a
// real download, then an inspection of the filesystem it touched.
func TestDownloadedMediaCreatesNoPicoPath(t *testing.T) {
	// A private TMPDIR, so the walk sees only what this test caused and a
	// developer's own /tmp cannot mask or fake a result.
	tempRoot := t.TempDir()
	t.Setenv("TMPDIR", tempRoot)

	// DownloadFile uses http.DefaultTransport when no proxy is configured. A
	// deterministic in-memory transport drives that real code path without
	// requiring a loopback listener (some test sandboxes deny all sockets).
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"image/png"}},
			Body:       io.NopCloser(strings.NewReader("not really a png, but it is bytes on disk")),
			Request:    req,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	localPath := utils.DownloadFile("https://download.invalid/photo.png", "photo.png",
		utils.DownloadOptions{LoggerPrefix: "test"})
	if localPath == "" {
		t.Fatal("the download helper returned no path, so this proves nothing")
	}
	if _, err := os.Stat(localPath); err != nil {
		t.Fatalf("the downloaded file is not on disk: %v", err)
	}

	// The streaming downloader used by skill installs owns a separate temp-file
	// prefix. Exercise it in the same private TMPDIR so the regression covers
	// both shipped download paths rather than only channel media.
	request, err := http.NewRequest(http.MethodGet, "https://download.invalid/skill.zip", nil)
	if err != nil {
		t.Fatal(err)
	}
	streamedPath, err := utils.DownloadToFile(t.Context(), &http.Client{
		Transport: http.DefaultTransport,
	}, request, 1024)
	if err != nil {
		t.Fatalf("streaming download failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(streamedPath) })
	if _, err := os.Stat(streamedPath); err != nil {
		t.Fatalf("the streamed file is not on disk: %v", err)
	}

	// The directory it chose must be the PocketClaw one, by name.
	if !strings.Contains(localPath, media.TempDirName) {
		t.Fatalf("media landed outside the canonical cache: %q", localPath)
	}

	if offenders := walkForPicoPaths(t, tempRoot); len(offenders) > 0 {
		t.Fatalf("PocketClaw created %d path(s) named after Pico: %v",
			len(offenders), offenders)
	}
}

// The constant itself, asserted directly so a rename cannot be partial.
func TestTempDirNameIsNotAPicoIdentity(t *testing.T) {
	if strings.Contains(strings.ToLower(media.TempDirName), "pico") {
		t.Fatalf("the runtime media cache is named after Pico: %q", media.TempDirName)
	}
	if !strings.Contains(media.TempDirName, "pocketclaw") {
		t.Fatalf("the runtime media cache is not named after the product: %q",
			media.TempDirName)
	}
}

// The legacy directory is deleted, not migrated and not recreated.
func TestRetireLegacyTempDirRemovesTheOldCache(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("TMPDIR", tempRoot)

	legacy := media.LegacyTempDir()
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(legacy, "leftover.png")
	if err := os.WriteFile(stale, []byte("stale attachment"), 0o600); err != nil {
		t.Fatal(err)
	}

	media.RetireLegacyTempDir()

	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("the legacy media cache survived: %v", err)
	}
	// And nothing was carried across: it is a cache, not user data.
	if _, err := os.Stat(filepath.Join(media.TempDir(), "leftover.png")); err == nil {
		t.Fatal("the legacy cache was migrated; it should be discarded")
	}
}

// Retiring is safe to call when there is nothing to retire, which is every
// fresh install.
func TestRetireLegacyTempDirIsSafeWithNoLegacyDirectory(t *testing.T) {
	tempRoot := t.TempDir()
	t.Setenv("TMPDIR", tempRoot)

	media.RetireLegacyTempDir()

	if offenders := walkForPicoPaths(t, tempRoot); len(offenders) > 0 {
		t.Fatalf("retiring created Pico paths: %v", offenders)
	}
}
