## What and why

<!-- What this changes, and the reason. The diff already shows the how. -->

## Verification

<!--
How you know it works. A green build is not verification on this project —
say what you actually ran or observed.
-->

- [ ] `python3 tool/release_gate.py --verify-source --release-class production` passes
- [ ] `flutter analyze lib/ test/` and `flutter test` pass (if Dart changed)
- [ ] Core rebuilt and re-staged (if anything under `core/src/` changed)
- [ ] Physically checked on a device (if behaviour changed)

## Notes

<!--
Anything a reviewer would otherwise have to discover: a legacy name you kept
and why, a decision you would like vetoed cheaply, a follow-up you deliberately
did not do.
-->
