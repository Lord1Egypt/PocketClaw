# PocketClaw Core source patch

The PicoClaw Core source is not vendored into this repository. It is kept as a
reviewed reference checkout at:

    /home/lordegypt/PocketCLaw/.upstream/picoclaw-core-v0.3.1

pinned at upstream commit `2cf030d2fd3b871d7ec17e3be34c24688aac76da`
(release `v0.3.1`).

`pocketclaw-core-v0.3.1.patch` is the complete set of PocketClaw modifications
applied on top of that commit. It covers the Android active-network DNS
integration, the model-API changes, and the PocketClaw user-facing wording in
the embedded web runtime and the seeded onboarding workspace.

`pkg/androiddns/` is a new package and is therefore untracked in the reference
checkout; it is not part of this patch and must be preserved in that checkout.

## Rebuilding the Android arm64 runtime binaries

    C=/home/lordegypt/PocketCLaw/.upstream/picoclaw-core-v0.3.1
    export PATH="/home/lordegypt/PocketCLaw/.tooling/pnpm/node_modules/.bin:$PATH"
    export GOTOOLCHAIN=auto      # the Makefile pins GOTOOLCHAIN=local; go1.25.11
                                 # is already in the Core-local module cache
    cd $C
    make build-android-arm64          VERSION=v0.3.1 GIT_COMMIT=2cf030d2
    make build-launcher-android-arm64 VERSION=v0.3.1 GIT_COMMIT=2cf030d2

Outputs, which are copied into `android/app/src/main/jniLibs/arm64-v8a/`:

| Build output | Installed as |
| --- | --- |
| `build/picoclaw-android-arm64` | `libpicoclaw.so` |
| `build/picoclaw-launcher-android-arm64` | `libpicoclaw-web.so` |

Both targets carry `-s -w`, so the shipped binaries are stripped. Always build
through these Makefile targets: running `make -C web build-android-arm64`
directly omits the root `LDFLAGS` and produces an unstripped binary.

Use `pnpm lint` to check the frontend. Do not use `pnpm check` — it runs
`prettier --write` over the whole tree and rewrites unrelated files.
