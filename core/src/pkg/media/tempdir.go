package media

import (
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/logger"
)

// TempDirName is the runtime media cache directory PocketClaw creates.
//
// Zero-Pico. This was `picoclaw_media`, and it is the one Pico identity the
// product was still actively *minting*: every Telegram, Matrix, Feishu, WeCom
// and TTS download created it, so a physical log showed new files appearing in
// the device's cache under a directory named `picoclaw_media`. That is not
// provenance, a legacy read or a migration -- it is this product writing
// someone else's name.
//
// It escaped the namespace guard because that guard treats `core/src` as
// vendored upstream except for named paths, and this file was not one of them.
// The guard now names this file, so a regression here fails the source gate.
const TempDirName = "pocketclaw_media"

// LegacyTempDirName is the pre-migration name, kept only so the directory an
// older build created can be deleted. Never written.
const LegacyTempDirName = "picoclaw_media"

// TempDir returns the shared temporary directory used for downloaded media.
func TempDir() string {
	return filepath.Join(os.TempDir(), TempDirName)
}

// LegacyTempDir returns the directory an older build wrote media into.
func LegacyTempDir() string {
	return filepath.Join(os.TempDir(), LegacyTempDirName)
}

// RetireLegacyTempDir deletes the media cache an older build created.
//
// Safe to delete outright rather than migrate: this is a cache of downloaded
// attachments under the OS temp directory, every entry is already referenced by
// absolute path in a session that has ended, and the media store re-downloads
// what it still needs. Nothing here is user data.
//
// Best effort, and deliberately quiet about the ordinary case: a fresh install
// has no legacy directory and must not log about one.
func RetireLegacyTempDir() {
	legacy := LegacyTempDir()
	if legacy == TempDir() {
		return
	}
	info, err := os.Stat(legacy)
	if err != nil || !info.IsDir() {
		return
	}
	if err := os.RemoveAll(legacy); err != nil {
		logger.WarnCF("media", "Could not remove the legacy media cache", map[string]any{
			"error": err.Error(),
		})
		return
	}
	logger.InfoC("media", "Removed the legacy media cache directory")
}
