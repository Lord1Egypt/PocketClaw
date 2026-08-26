# Services

PocketClaw's runtime source lives in this repository. Services it *talks to*
over the network are deployed separately and live in their own repositories.

## Telegram onboarding

**<https://github.com/Lord1Egypt/PocketClaw-Telegram-Setup>** — public, MIT.

The manager service behind PocketClaw's Telegram managed-bot onboarding. It
issues pairing sessions, receives Telegram's `managed_bot` webhook, and
delivers each child bot token once to the PocketClaw install that asked for it.

It started life in this repository at `services/telegram-onboarding/` and was
extracted on 2026-08-26, when it was reworked for serverless deployment. Two
things forced the rework, and both are why it could not stay as it was:

- Pairing state was a process-local Go map. On Vercel the request that creates
  a pairing, the webhook that completes it, and the request that collects the
  token each land in a different function instance, so state moved to a
  Redis-compatible store with atomic `SET NX` and `GETDEL`.
- Updates arrived by `getUpdates` long-polling, which needs a process that
  stays alive. It is now a webhook, gated by the secret Telegram echoes.

### Why it is not vendored here

It is infrastructure, not application source. The APK does not build from it
and does not contain it — the app holds only a public HTTPS base URL, supplied
at build time as `POCKETCLAW_ONBOARDING_BASE_URL`. The self-contained
source-of-truth rule covers what builds the APK, which is `core/src/` and
`lib/`; see `DECISIONS.md`.

It is also useful to other people on its own terms, which a directory inside an
Android app's repository is not.

### The contract this app depends on

| Endpoint | Purpose |
| --- | --- |
| `POST /telegram/pairings` | start a pairing; returns the deep link, QR payload, and poll token |
| `GET /telegram/pairings/{id}` | poll state: `pending`, `created`, `ready`, `failed`; never returns a token |
| `POST /telegram/pairings/{id}/token` | collect the child bot token, exactly once |

The Flutter client for it is `lib/src/telegram/`. An expired, unknown, or
wrongly-authenticated pairing answers `404`, which the client reads as expired.
