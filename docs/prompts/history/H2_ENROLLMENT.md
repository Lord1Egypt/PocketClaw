# H2 — Developer production signer enrollment

> **RECONSTRUCTED OPERATING RECORD — not the original prompt.** Rebuilt from Git
> history, public certificate enrollment, tests, and signing documentation.
> Private keystore location and passwords are deliberately absent.

- **Status:** CLOSED, 2026-09-10.
- **Starting commit:** `fb38c7d3c3fd31420c45a1977e1c1425ede3a378`.
- **Ending commit:** `78d33fd5b179dd52c7cd8118a5d23ec19c8368ec`.
- **Purpose:** Create the owner-held developer production key, enroll exactly
  one public certificate digest, and verify that the ceremony never records
  private material.

## Reconstructed scope and invariants

- Owner alone handles keystore and passwords outside Git.
- Track only the public certificate SHA-256:
  `176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`.
- Enroll exactly one digest; certificate subject text is not identity evidence.
- Do not build, install, publish, accept, merge, tag, or advance the baseline.

## Outcome and defect

The key was created and the public digest enrolled. The ceremony exposed a
directly related helper defect: hidden `keytool` stderr suppressed the password
prompt and allowed an empty fingerprint pipeline to exit successfully. Commit
`78d33fd` preserved stderr, checked fingerprint shape, added a reusable
`--print-fingerprint` path, and added a disposable-keystore regression test.

No production artifact existed at enrollment closeout. Version, baseline, Core,
runtime payloads, devices, releases, and protected refs were unchanged.

- **Next authorized milestone at closeout:** H2 private production-signing
  validation by owner-only hidden input.
