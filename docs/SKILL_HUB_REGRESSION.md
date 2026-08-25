# Skill Hub Regression Record

## Physical-device state

Status: **PASS**. The previously observed `Registry search is currently unavailable` message is classified as **RESOLVED BY ANDROID DNS FIX**.

After the Android active-network DNS integration, physical-device verification confirmed that Skill Hub opens, ClawHub is reachable, a `Crypto` search returns 20 results, metadata and URLs render, and install actions are visible.

## Milestone B boundary

Skill Hub is Core/WebView functionality; no registry configuration or architecture was changed in this milestone. The Dart/Android foundation has no independent registry client to unit-test without duplicating the Core behavior.

## Required smoke regression

1. Open Skill Hub.
2. Search the ClawHub registry.
3. Confirm results, metadata, URLs, and install action render.
4. Review author, source URL, instructions, capabilities, credentials, and code before choosing an install action.
5. Do not install a third-party skill automatically; explicit user approval is required.
