# Contributing to PocketClaw

Thanks for taking the time. This project has a few conventions that are easier
to read once than to discover by trial.

## Before you start

Read [`PROJECT_STATE.md`](PROJECT_STATE.md) for where things actually stand, and
[`SESSION_HANDOFF.md`](SESSION_HANDOFF.md) for the short list of things that
look like cleanup but are load-bearing. The second file exists because several
of them have been "tidied up" before.

## The rules that matter

**Run the gate, not just the tests.**

```bash
python3 tool/release_gate.py --verify-source --release-class production
```

`tool/release_gate.py` is the single definition of what "releasable" means. CI
runs the same command, so it cannot disagree with your machine.

**Any change under `core/src/` needs a Core rebuild.** The shipped binaries are
committed artifacts. Editing Go or frontend source without re-running
`./core/build-android-arm64.sh` leaves the staged binaries describing code that
is no longer in the tree, and the gate will say so.

**Don't remove a legacy name without checking what reads it.** PocketClaw was
renamed from an upstream project, and the old identifiers that remain are how an
older installation is migrated rather than stranded. `tool/no_active_pico.py`
will tell you which category any given occurrence is in and why it is allowed.

**Version and baseline move separately.** `pubspec.yaml` carries the candidate
version. `android/release-baseline.properties` carries the last versionCode that
passed on a real device, and it advances by hand, only in the commit that
records that acceptance.

## Style

Match the surrounding code — its naming, its idiom, and its comment density.
Comments should state a constraint the code cannot show; the code already says
what it does.

## Pull requests

Keep the diff to one concern. Explain *why* in the description; the diff covers
what. If you changed behaviour, say how you verified it — a green build is not
verification on this project.

## Security

Do not open a public issue for a vulnerability. See [`SECURITY.md`](SECURITY.md).
