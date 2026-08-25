# PocketClaw Telegram Onboarding Service

Turns "create a bot in BotFather, copy the token, paste it into the app" into
"tap Open Telegram, confirm, done".

This service is PocketClaw's own. Its source lives in this repository, it has
zero external Go dependencies, and it depends on no third-party onboarding
provider at runtime — only on Telegram itself.

## Architecture

```
PocketClaw app                 this service                Telegram
     │                              │                          │
     │ POST /telegram/pairings      │                          │
     ├─────────────────────────────>│                          │
     │  pairing_id, poll_token,     │                          │
     │  deep_link, qr_payload       │                          │
     │<─────────────────────────────┤                          │
     │                              │                          │
     │  open deep_link / show QR ───┼─────────────────────────>│
     │                              │      user confirms       │
     │                              │   "Create bot" screen    │
     │                              │                          │
     │                              │<── Update.managed_bot ───┤
     │                              │    getManagedBotToken    │
     │                              ├─────────────────────────>│
     │ GET  /telegram/pairings/{id} │                          │
     ├─────────────────────────────>│                          │
     │  state: ready                │                          │
     │<─────────────────────────────┤                          │
     │ POST /telegram/pairings/{id}/token                      │
     ├─────────────────────────────>│  (single use, then the   │
     │  bot_token, bot_username,    │   session is destroyed)  │
     │  owner_user_id               │                          │
     │<─────────────────────────────┤                          │
     │                              │                          │
   writes the token into Core's config.json, restarts the channel
```

The app never receives the manager bot's credentials. The service never
receives anything about the user's conversations, models, or provider keys.

## Telegram capabilities used

All of this is official Telegram, introduced in **Bot API 9.6 (2026-04-03)**.
Nothing here is inferred from another project's implementation.

| Telegram surface | Used for |
| --- | --- |
| `https://t.me/newbot/{manager}/{suggested}[?name={name}]` | opening the pre-filled creation screen |
| `User.can_manage_bots` (Boolean) | verifying the manager bot at startup |
| `Update.managed_bot` → `ManagedBotUpdated{user, bot}` | learning that a child bot was created, and by whom |
| `getManagedBotToken(user_id)` → String | retrieving the child bot's token |

`ManagedBotUpdated.user` is the Telegram user who created the bot. PocketClaw
uses it as the child bot's `allow_from` entry, so a freshly paired bot answers
only its owner.

Telegram still shows its own confirmation screen and the user still presses
Create. PocketClaw pre-fills the name and username; it does not and cannot
create a bot without that confirmation.

Two Telegram methods exist that this service deliberately does not call:
`replaceManagedBotToken` (rotation is not part of onboarding) and
`setManagedBotAccessSettings` (PocketClaw restricts access through Core's own
`allow_from`, not through Telegram's access list).

## One-time operator setup

1. Create the manager bot with [@BotFather](https://t.me/BotFather).
   The intended identity is display name **PocketClaw Setup**, username
   **@PocketClawSetupBot**. If that username is taken, pick another
   PocketClaw-owned one and set `TELEGRAM_MANAGER_BOT_USERNAME` to it — nothing
   in the code hardcodes the username.
2. Open BotFather's mini app, select that bot, and enable **Bot Management
   Mode**. This is what makes Telegram set `can_manage_bots` on the bot.
3. Copy the bot token from BotFather.
4. Configure this service with the token and username (see below).
5. Start the service. It calls `getMe` at startup and **refuses to start** if
   `can_manage_bots` is false or the username does not match the token, rather
   than issuing links that could never resolve.

Until step 2 is done, the deep links this service produces will open Telegram
but Telegram will not offer to create a bot.

## Configuration

Copy `.env.example` to `.env` and fill it in. Required:

| Variable | Secret? | Meaning |
| --- | --- | --- |
| `TELEGRAM_MANAGER_BOT_TOKEN` | **yes** | manager bot token from BotFather |
| `TELEGRAM_MANAGER_BOT_USERNAME` | no | manager bot `@username`, without the `@` |
| `PUBLIC_ONBOARDING_BASE_URL` | no | HTTPS base URL the app is pointed at |

Optional: `LISTEN_ADDR`, `PAIRING_TTL_SECONDS`, `RATE_LIMIT_BURST`,
`RATE_LIMIT_PER_SECOND`, `TRUST_PROXY_HEADER`.

`TELEGRAM_MANAGER_BOT_TOKEN` is a **server secret**. It must never appear in
the APK, in Flutter assets, in Core source, in this repository, in
`PROJECT_STATE.md`, or in a log line. The service redacts it from every error
string it produces, including transport errors that quote the request URL.

`TRUST_PROXY_HEADER` must stay `false` unless a reverse proxy in front of this
service *overwrites* `X-Forwarded-For`. If it merely appends, clients can forge
the header and evade the rate limit.

## Pairing API

### `POST /telegram/pairings`

Creates a session. Rate-limited per client.

```json
{
  "pairing_id": "<32 hex chars>",
  "poll_token": "<64 hex chars>",
  "suggested_username": "pocketclaw_k7m2x9aa_bot",
  "suggested_name": "PocketClaw Agent",
  "deep_link": "https://t.me/newbot/PocketClawSetupBot/pocketclaw_k7m2x9aa_bot?name=PocketClaw%20Agent",
  "qr_payload": "https://t.me/newbot/PocketClawSetupBot/pocketclaw_k7m2x9aa_bot?name=PocketClaw%20Agent",
  "expires_at": "2026-08-26T12:34:56Z",
  "poll_interval_seconds": 2
}
```

`poll_token` is returned here and never again. `qr_payload` is deliberately
identical to `deep_link`: Telegram defines no separate QR form, and the QR must
carry no secret.

### `GET /telegram/pairings/{pairing_id}`

Requires `Authorization: Bearer <poll_token>`. **Never returns a token.**

```json
{
  "pairing_id": "...",
  "state": "pending | created | ready | expired | failed",
  "reason": "token_retrieval_failed",
  "bot_username": "pocketclaw_k7m2x9aa_bot",
  "owner_user_id": 555,
  "expires_at": "2026-08-26T12:34:56Z"
}
```

| State | Meaning |
| --- | --- |
| `pending` | link issued, Telegram has reported nothing yet |
| `created` | the child bot exists; its token is being retrieved |
| `ready` | the token is held, awaiting its single delivery |
| `expired` | the session outlived its TTL |
| `failed` | the session cannot complete; `reason` says why, without secrets |

An expired or swept session answers `404`, the same as an unknown one.

### `POST /telegram/pairings/{pairing_id}/token`

Requires `Authorization: Bearer <poll_token>`. Delivers the child token
**exactly once**, then destroys the session.

```json
{
  "bot_token": "9001:...",
  "bot_user_id": 9001,
  "bot_username": "pocketclaw_k7m2x9aa_bot",
  "owner_user_id": 555
}
```

Polling and collection are separate endpoints on purpose. Polling is
idempotent and happens many times; collection is a one-shot transition that
wipes the server's copy. Folding the token into the poll response would mean
either repeating it on every poll or making polling destructive.

### `GET /healthz`

Reports `status`, the configured `manager_username`, and the live pairing
count. Carries no secrets.

## Security assumptions

- **Pairing IDs and poll tokens are independent.** 16 and 32 bytes from
  `crypto/rand` respectively, hex-encoded. Neither is derived from the other,
  and neither is sequential.
- **Poll tokens are stored hashed.** Only the SHA-256 is kept, compared in
  constant time. A memory dump or an accidental log of the store yields nothing
  usable.
- **A wrong token is indistinguishable from an unknown pairing.** Both answer
  `404`, so live pairing IDs cannot be enumerated.
- **A failed collection attempt does not burn the delivery.** Only a
  correctly authenticated call consumes the token.
- **No secret is ever in a URL, a QR code, or a deep link.** The QR contains
  only the public Telegram creation link.
- **The child token exists server-side for seconds.** It is held between
  retrieval and delivery and wiped on delivery, on failure, and on expiry.
- **Serve over HTTPS only.** The pairing API carries a bearer token and,
  once, a bot token. Terminate TLS in front of this service or run it behind a
  platform that does.

### Correlation, and its one limitation

An incoming `managed_bot` update is matched to a pairing by the child bot's
username, which is why each pairing suggests a unique random one. If the user
edits the suggested username on Telegram's confirmation screen, the update
matches no pairing and the session stays `pending` until it expires. That is a
deliberate fail-closed choice: matching loosely — say, by "the only pending
pairing" — would let one user's bot be delivered into another user's app. The
app surfaces the expiry and offers a retry and the manual fallback.

## Privacy

The service holds, in memory only, for at most the pairing TTL (10 minutes by
default):

| Stored | Why | Deleted |
| --- | --- | --- |
| pairing ID, suggested username/name, deep link, timestamps | to issue and match the pairing | on delivery or expiry |
| SHA-256 of the poll token | to authenticate polling | on delivery or expiry |
| Telegram owner user ID, child bot ID and username | to configure the child bot's allow-list | on delivery or expiry |
| the child bot token | to hand it to the app once | **on delivery**, or on failure or expiry |

There is no database and nothing is written to disk. The service never sees
chat contents, AI messages, provider API keys, or PocketClaw workspace data.
Restarting the service drops all live pairings; in-flight users retry.

## Scaling

Updates are ingested by long-polling `getUpdates`, not by a webhook. That needs
no inbound reachability for Telegram, no webhook secret, and no TLS coupling.

The consequence is a hard constraint: **run exactly one instance per manager
bot token.** Telegram permits only one `getUpdates` consumer at a time, and
pairings live in that instance's memory. For a service whose sessions last ten
minutes and whose state is disposable, one instance behind a restart policy is
the right shape. If a future deployment needs several, that change is a webhook
plus shared state, and both must land together.

## Local development

```bash
cd services/telegram-onboarding
go test ./...            # no network, no secrets, no Telegram account needed
go build ./...
```

Every test drives a fake Telegram. There is no test that requires a real
manager bot or a production secret.

To run the server you do need a real manager bot, because it verifies itself at
startup:

```bash
set -a && source .env && set +a
go run ./cmd/server
curl -s localhost:8080/healthz
```

## Deployment

The service is a single static Go binary with no dependencies, no database, and
no disk state. It listens on `LISTEN_ADDR` and needs outbound HTTPS to
`api.telegram.org`.

```bash
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o pocketclaw-onboarding ./cmd/server
```

That runs on a container host, a VPS under systemd, or any platform that can
run a long-lived process. It is not tied to a hosting vendor. What a deployment
must provide:

- HTTPS termination in front of the service.
- The two required environment variables, from a secret store rather than a
  file baked into an image.
- A restart policy. Losing the process loses live pairings, which is safe —
  users retry — but a stopped process onboards nobody.
- Exactly one running instance per manager bot token, per "Scaling" above.
- `GET /healthz` as the liveness and readiness probe.

Finally, point the PocketClaw app at the deployment's public base URL.
