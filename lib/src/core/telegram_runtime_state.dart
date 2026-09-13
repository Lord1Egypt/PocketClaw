import 'status_snapshot.dart';

/// What Telegram is actually doing, as opposed to what was configured.
///
/// PC-DEF-027. The Telegram surface used to report "Connected" whenever a bot
/// username had been written to SharedPreferences, so a channel Core had
/// refused -- "telegram requires exactly one paired numeric owner" -- still
/// presented as working. Stored state is not evidence: a username says a
/// pairing once happened, not that a channel is running now.
///
/// Core already publishes the authoritative facts as three separate booleans
/// per channel (`configured`, `started`, `running`) and deliberately publishes
/// no reachability field, because nothing probes the network. So this
/// distinguishes exactly what Core knows and no more.
enum TelegramRuntimeState {
  /// No Telegram channel in the runtime at all.
  notConfigured,

  /// Core holds the channel but it is not running -- typically disabled, or
  /// the gateway has not started it yet.
  configuredNotRunning,

  /// Core started the channel and it is running.
  ///
  /// This is what the product calls "Connected", and it means exactly that the
  /// Telegram runtime channel is running. It does NOT mean Telegram's servers
  /// were reached or that the token was independently verified: Core performs
  /// no such probe, so the UI must not imply one.
  running,

  /// Configured, but startup failed: Core holds it and it never started.
  error,
}

/// Derives Telegram state from Core's reported channels.
///
/// [channels] being null means the runtime has not reported yet -- the gateway
/// may be stopped or still starting. That is deliberately not an error and
/// deliberately not running. Callers pass `serviceManager.statusSnapshot
/// ?.channels`, so a missing snapshot and a missing channel converge on the
/// same answer without either pretending to be healthy.
TelegramRuntimeState resolveTelegramRuntimeState(List<StatusChannel>? channels) {
  if (channels == null) return TelegramRuntimeState.notConfigured;

  StatusChannel? telegram;
  for (final channel in channels) {
    if (channel.name.toLowerCase() == 'telegram') {
      telegram = channel;
      break;
    }
  }
  if (telegram == null || !telegram.configured) {
    return TelegramRuntimeState.notConfigured;
  }
  if (telegram.running) return TelegramRuntimeState.running;

  // Configured and not running splits on whether a start was ever attempted
  // and returned successfully. A channel that started but is no longer running
  // has stopped; one that never started failed to.
  return telegram.started
      ? TelegramRuntimeState.configuredNotRunning
      : TelegramRuntimeState.error;
}

/// Whether the product may show its "Connected" wording.
///
/// The single place that answers it, so no surface can invent a second
/// definition from a token, a username or an `enabled` flag.
bool telegramMayReportConnected(TelegramRuntimeState state) =>
    state == TelegramRuntimeState.running;
