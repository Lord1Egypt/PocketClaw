# PocketClaw Core

This directory holds PocketClaw's Core runtime source and the scripts that turn
it into the two Android arm64 binaries the app ships.

## Where the source lives

    core/src/

That is the canonical build source. A normal clone of this repository contains
everything needed to build the PocketClaw runtime — there is no submodule, no
second repository to clone, and no external checkout to prepare.

    core/src/                        canonical Core source (upstream + PocketClaw changes)
    core/build-android-arm64.sh      canonical Core build
    core/verify-no-external-source.sh proves the build needs no external checkout
    core/regen-upstream-patch.sh     regenerates the upstream-vs-PocketClaw patch
    core/pocketclaw-core-v0.3.1.patch the divergence from upstream, as provenance

## Upstream origin

`core/src/` originates from PicoClaw and remains MIT-licensed upstream work
with PocketClaw modifications layered on top.

| | |
| --- | --- |
| Original project | PicoClaw |
| Original release | `v0.3.1` |
| Original upstream commit | `2cf030d2fd3b871d7ec17e3be34c24688aac76da` |
| Upstream repository | <https://github.com/sipeed/picoclaw> |
| Adopted by PocketClaw | 2026-08-24 |

The distinction that matters:

    SOURCE LOCATION:      PocketClaw repository
    ORIGINAL PROVENANCE:  PicoClaw v0.3.1

Upstream copyright headers are preserved, `core/src/LICENSE` is the upstream
MIT license kept in place, and the same text is reproduced at
`licenses/picoclaw-core-MIT.txt`. Attribution lives in
`THIRD_PARTY_NOTICES.md`, and the baseline provenance in
`UPSTREAM_BASELINE.md`. Nothing upstream-authored is claimed as PocketClaw's
own work.

### Deliberate omissions from the vendored tree

- `assets/` — 13 MB of upstream README screenshots and marketing GIFs. Not a
  build input, and not PocketClaw's branding to carry.
- `pkg/seahorse/.omc/` — an upstream developer's tool-state file that was
  committed by accident. It contains no code and leaks an upstream
  contributor's home directory path.

Neither is referenced by any build target. Both are excluded on both sides when
the provenance patch is regenerated, so they never appear as spurious
deletions.

## PocketClaw modifications

`core/src/` is upstream `v0.3.1` plus every change PocketClaw needs in
production:

- Android active-network DNS integration — `pkg/androiddns/`, wired through
  `cmd/picoclaw/dns_noresolv.go` and `PICOCLAW_DNS_SERVER`. The Go Android
  localhost fallback (`[::1]:53`) is never used and no public resolver is
  hardcoded.
- Provider catalog extensions — `Category` and `DocumentationURL` metadata
  across the preset table in `pkg/providers/provider_metadata.go`, plus the
  provider-first picker in `web/frontend/src/components/models/`.
- Model discovery fixes — the Gemini fetch branch and the OpenCode fetch
  branch in `web/backend/api/models.go`.
- OpenCode Zen and OpenCode Go presets with per-model protocol routing —
  `pkg/providers/opencode_routing.go`.
- A generic Responses provider — `pkg/providers/openai_responses/`.
- Opt-in bearer auth for the Anthropic Messages surface —
  `pkg/providers/anthropic_messages/`.
- MQTT `/pocketclaw` fresh-install default that never rewrites a configured
  prefix — `pkg/channels/mqtt/`.
- Seeded onboarding workspace and user-facing wording — `workspace/`,
  `cmd/picoclaw/internal/onboard/`, `web/backend/i18n.go`, and the frontend
  locale files.
- `-trimpath` on the Android arm64 build targets, so release binaries carry no
  developer-machine paths. See "Build path privacy" below.
- `/onboard` anchored in `core/src/.gitignore`. Upstream's bare `onboard` rule
  was meant for a generated root-level artifact but also swallowed
  `cmd/picoclaw/internal/onboard/`, which carries PocketClaw changes.

## Building the Android arm64 runtime

    core/build-android-arm64.sh

That is the whole procedure. It builds both binaries from `core/src` and
installs them into `android/app/src/main/jniLibs/arm64-v8a/`.

| Build output | Installed as |
| --- | --- |
| `core/src/build/picoclaw-android-arm64` | `libpicoclaw.so` |
| `core/src/build/picoclaw-launcher-android-arm64` | `libpicoclaw-web.so` |

Under the hood it runs the two root Makefile targets, which is the only
supported path:

    cd core/src
    make build-android-arm64          VERSION=v0.3.1 GIT_COMMIT=2cf030d2
    make build-launcher-android-arm64 VERSION=v0.3.1 GIT_COMMIT=2cf030d2

`VERSION` and `GIT_COMMIT` are passed explicitly on purpose. Left to itself the
Makefile derives them from `git describe`, which now resolves against
PocketClaw's history and would stamp a PocketClaw tag onto upstream-versioned
Core.

Always build through the **root** Makefile targets. Running
`make -C web build-android-arm64` directly drops the root `LDFLAGS` and
produces an unstripped ~34 MB web runtime.

### Release flags

Both targets compile with:

    GOOS=android GOARCH=arm64 go build -trimpath -tags stdjson \
      -ldflags "-X ...config.Version=v0.3.1 \
                -X ...config.GitCommit=2cf030d2 \
                -X ...config.BuildTime=<timestamp> \
                -X ...config.GoVersion=go1.25.11 \
                -s -w"

`-s -w` is what makes the shipped binaries stripped. `CGO_ENABLED=0` comes from
the Makefile. The expected release profile is roughly 37.2 MB for
`libpicoclaw.so` and 24.6 MB for `libpicoclaw-web.so`; both are
`ELF 64-bit LSB pie executable, ARM aarch64 ... stripped`.

### Build path privacy

`-trimpath` is not optional. Without it, Go embeds absolute source paths for
every compiled file, and the binaries carried thousands of
`/home/lordegypt/...` strings that could surface on the user-facing Logs
screen. The build script asserts the count is zero and fails the build if it
regresses. Source provenance belongs in this file, not in a user's log output.

### Verifying the hashes

    sha256sum android/app/src/main/jniLibs/arm64-v8a/libpicoclaw*.so

The build script prints size and SHA-256 for both binaries as it installs them.
Record the values in `UPSTREAM_BASELINE.md`. Hashes legitimately change when
the source or the build timestamp changes; sizes should stay near the profile
above.

## External toolchain prerequisites

These are build tools, not application source. A clean clone needs them
installed but does not need any additional source repository.

| Tool | Used for | This machine |
| --- | --- | --- |
| Go (per `core/src/go.mod`, currently 1.25.11) | Core binaries | resolved from `GOMODCACHE` with `GOTOOLCHAIN=auto` |
| Node + pnpm | embedded web frontend | `/home/lordegypt/PocketCLaw/.tooling/pnpm/node_modules/.bin` |
| JDK 17 | Android build | `/home/lordegypt/PocketCLaw/.tooling/jdk-17` |
| Android SDK/NDK | Android build | `/home/lordegypt/PocketCLaw/.tooling/android-sdk` |
| Flutter | the app itself | `/home/lordegypt/PocketCLaw/.tooling/flutter` |

The Go build and module caches live at
`/home/lordegypt/PocketCLaw/.tooling/go/`. They are caches, not source, and the
build script points `GOCACHE`/`GOMODCACHE` at them. Override `GO_CACHE_ROOT`
and `PNPM_BIN_DIR` if your toolchain lives elsewhere; a fresh machine can also
let Go use its defaults and download from `go.sum`.

The frontend bootstraps itself: `make build-frontend` runs
`pnpm install --frozen-lockfile` when `node_modules` is absent.

## Packaging into the APK

Build Core first, then package:

    export JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17
    export GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean

    cd /home/lordegypt/PocketClaw-App/android
    ./gradlew :app:assembleRelease -Ptarget-platform=android-arm64

`-Ptarget-platform=android-arm64` is required. Dropping it produces a ~50 MB
multi-ABI APK.

### The release guard

`android/app/build.gradle.kts` fails the release build if the APK is missing
any of:

    lib/arm64-v8a/libdartjni.so
    lib/arm64-v8a/libpicoclaw.so
    lib/arm64-v8a/libpicoclaw-web.so

Do not remove this. A missing `libdartjni.so` once shipped silently for days
and produced a black screen on the device while the build stayed green.

**Recovery procedure** if `libdartjni.so` goes missing: the cause is a poisoned
CMake configure cache in the `jni` pub package, which records the first ABI it
configured and then refuses to produce the others. It lives outside this
project and a clean build of this repository will not clear it.

    rm -rf ~/.pub-cache/hosted/pub.dev/jni-*/android/.cxx/
    rm -rf android/app/build android/build
    cd android && ./gradlew :app:assembleRelease -Ptarget-platform=android-arm64

Then confirm the guard passes and the APK really contains all three libraries:

    unzip -l <apk> | grep 'lib/arm64-v8a/'

## The upstream provenance patch

`core/pocketclaw-core-v0.3.1.patch` is the difference between upstream PicoClaw
`v0.3.1` and PocketClaw's Core baseline. It is no longer how the source is
assembled — `core/src/` is the source — but it stays as provenance, so a future
upstream review can see PocketClaw's divergence at a glance.

Regenerate it after any change under `core/src/`:

    core/regen-upstream-patch.sh

That fetches upstream `2cf030d2` into a temporary directory for review only and
diffs it against `core/src/`. Pass a local PicoClaw checkout to work offline:

    core/regen-upstream-patch.sh /path/to/picoclaw

Verify without rewriting:

    core/regen-upstream-patch.sh && git diff --stat core/pocketclaw-core-v0.3.1.patch

An empty diff means the patch on disk matches the vendored source.

## Why external `.upstream` checkouts are not build dependencies

A reference checkout may still exist at
`/home/lordegypt/PocketCLaw/.upstream/picoclaw-core-v0.3.1`. It is historical
and is used for upstream review only. No build target, script, or Gradle task
reads it.

Prove it at any time:

    core/verify-no-external-source.sh

That renames the external checkout out of the way, runs the full Core build,
and restores the rename afterwards. It never deletes anything.

## How future upstream reviews work

Upstream is an update source, never a production build dependency. The
checkpoint of record is `UPSTREAM_TRACKING.md`.

1. Read the last reviewed-through commit from `UPSTREAM_TRACKING.md`.
2. Fetch upstream temporarily, for review only.
3. Diff `last_reviewed..new_upstream`.
4. Identify the useful changes.
5. Adapt the selected ones into `core/src/`.
6. Test, build, and physically verify.
7. Regenerate the provenance patch and update `UPSTREAM_TRACKING.md`.

Never restart the analysis from `v0.3.1` once later commits have been reviewed.

## Tests

    cd core/src
    make test        # Go suites plus the web backend/frontend suites
    cd web/frontend && pnpm lint

Do not run `pnpm check` — it runs `prettier --write` across the whole tree and
rewrites unrelated files.
