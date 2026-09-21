package logger

import "testing"

// The physical DEBUG logs carried absolute app-private paths. These pin that the
// install-specific prefix goes and the diagnostic remainder stays.
func TestAndroidPrivatePathsAreNormalized(t *testing.T) {
	t.Parallel()

	cases := map[string]struct{ in, want string }{
		"app-private cache": {
			in:   "/data/user/0/com.lord1egypt.pocketclaw/cache/tmp/pocketclaw_media/a.png",
			want: "<app-private>/cache/tmp/pocketclaw_media/a.png",
		},
		"legacy data/data form": {
			in:   "/data/data/com.lord1egypt.pocketclaw/no_backup/gateway-auth",
			want: "<app-private>/no_backup/gateway-auth",
		},
		"work profile user id": {
			in:   "/data/user/10/com.lord1egypt.pocketclaw/files/config.json",
			want: "<app-private>/files/config.json",
		},
		"device-encrypted root": {
			in:   "/data/user_de/0/com.lord1egypt.pocketclaw/files/x",
			want: "<app-private>/files/x",
		},
		"native library install path": {
			in:   "/data/app/~~kqZ9v1Q==/com.lord1egypt.pocketclaw-Ab3D==/lib/arm64/libpocketclaw.so",
			want: "<app-install>/lib/arm64/libpocketclaw.so",
		},
		"install path without the hashed parent": {
			in:   "/data/app/com.lord1egypt.pocketclaw-Ab3D==/base.apk",
			want: "<app-install>/base.apk",
		},
		"inside a sentence": {
			in:   "failed to open /data/user/0/com.lord1egypt.pocketclaw/files/a.db: no such file",
			want: "failed to open <app-private>/files/a.db: no such file",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := normalizeAndroidPaths(tc.in); got != tc.want {
				t.Fatalf("normalizeAndroidPaths(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// Shared external storage is public and the user chose the workspace path, so it
// must survive: normalizing it would make a log less useful for no privacy gain.
func TestSharedStoragePathsSurvive(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/storage/emulated/0/Download/pocketclaw/AGENT.md",
		"/sdcard/Download/pocketclaw/memory",
	} {
		if got := normalizeAndroidPaths(path); got != path {
			t.Fatalf("normalizeAndroidPaths(%q) = %q; shared storage must survive", path, got)
		}
	}
}

// Ordinary text must not be touched. The rules are anchored on literal /data
// roots precisely so prose cannot trip them.
func TestNonPathTextIsUntouched(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"model=deepseek-v4.1-flash tokens=512",
		"data/user is not an absolute path",
		"Turn blocked by configuration code=PC-E-AI-004",
	} {
		if got := normalizeAndroidPaths(text); got != text {
			t.Fatalf("normalizeAndroidPaths(%q) = %q; want unchanged", text, got)
		}
	}
}

// And it reaches structured log fields, which is where the paths actually were.
func TestPrivatePathsAreNormalizedInStructuredFields(t *testing.T) {
	t.Parallel()

	safe := sanitizeFieldsForLog(map[string]any{
		"path": "/data/user/0/com.lord1egypt.pocketclaw/cache/tmp/pocketclaw_media/b.jpg",
		"lib":  "/data/app/~~x==/com.lord1egypt.pocketclaw-y==/lib/arm64/libpocketclaw.so",
	})
	if got := safe["path"]; got != "<app-private>/cache/tmp/pocketclaw_media/b.jpg" {
		t.Fatalf("path field = %v", got)
	}
	if got := safe["lib"]; got != "<app-install>/lib/arm64/libpocketclaw.so" {
		t.Fatalf("lib field = %v", got)
	}
}

// Third-party DEBUG text takes a different entry point from structured fields.
// This is the boundary that receives downloader and SDK messages in practice.
func TestPrivatePathsAreNormalizedBeforeThirdPartyDebugLogs(t *testing.T) {
	t.Parallel()

	input := "downloaded /data/user/0/com.lord1egypt.pocketclaw/cache/tmp/pocketclaw_media/c.png"
	want := "downloaded <app-private>/cache/tmp/pocketclaw_media/c.png"
	got, keep := prepareThirdPartyLog(DEBUG, "download", input)
	if !keep || got != want {
		t.Fatalf("prepareThirdPartyLog() = %q, %v; want %q, true", got, keep, want)
	}
}
