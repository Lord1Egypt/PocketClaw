import 'status_snapshot.dart';

/// What Telegram is, as opposed to what was configured — with configuration
/// truth and runtime truth kept apart.
///
/// PC-DEF-027. The Telegram surface used to report "Connected" whenever a bot
/// username had been written to SharedPreferences, so a channel Core had
/// refused still presented as working. Stored state is not evidence that
/// anything is running.
///
/// The correction that followed matters just as much in the other direction: a
/// validly configured Telegram channel is still configured while the Gateway is
/// stopped, so runtime silence must not be reported as "not configured", and
/// the absence of a successful start is not proof that a start failed. Each
/// state below is only claimed when something actually says so.
enum TelegramRuntimeState {
  /// No enabled, valid Telegram configuration exists.
  ///
  /// A statement about configuration, so it is never concluded from runtime
  /// silence.
  notConfigured,

  /// Telegram is configured, but the runtime is not active or not observable.
  ///
  /// Covers every honest "configured but not running" case and deliberately
  /// does not distinguish them, because Core does not: the Gateway may be
  /// stopped, still starting, not yet reporting, or running with the channel
  /// disabled. None of those is a failure and none is Running.
  configuredRuntimeNotActive,

  /// Core reports the channel as running.
  ///
  /// This is what the product calls "Connected", and it means exactly that the
  /// Telegram runtime channel is running. It does NOT mean Telegram's servers
  /// were reached or the token independently verified: Core performs no such
  /// probe, so the UI must not imply one.
  running,

  /// Affirmative evidence that starting or applying the channel failed.
  ///
  /// Only ever produced from an explicit failure signal. Core exposes no
  /// per-channel error today, so nothing synthesizes this state — and in
  /// particular `started == false` does not, because "has not started" and
  /// "failed to start" are different facts.
  error,
}

/// Derives Telegram state from configuration truth and runtime truth.
///
/// [configuredAndValid] comes from persisted configuration — the only thing
/// that can answer whether Telegram is set up at all. The caller supplies it
/// from whatever configuration it holds; stored local state is legitimate
/// evidence *of configuration*, and never evidence of running.
///
/// [channels] is Core's runtime report. Null means the runtime has not been
/// observed — a stopped Gateway, a snapshot that has not arrived, or a status
/// read that failed. That is silence, not a verdict.
///
/// [startupError] is affirmative, sanitized evidence that the channel failed.
/// Absent means absent; it is never inferred.
TelegramRuntimeState resolveTelegramRuntimeState({
  required bool configuredAndValid,
  List<StatusChannel>? channels,
  String? startupError,
}) {
  final telegram = _telegramChannel(channels);

  // Running is affirmative evidence, and outranks everything: a channel that
  // reports running is running, whatever configuration was edited since.
  if (telegram != null && telegram.running) return TelegramRuntimeState.running;

  // Only a real failure signal produces an error.
  if (startupError != null && startupError.trim().isNotEmpty) {
    return TelegramRuntimeState.error;
  }

  // Configuration decides configured-ness. Runtime silence never does.
  if (!configuredAndValid) return TelegramRuntimeState.notConfigured;

  return TelegramRuntimeState.configuredRuntimeNotActive;
}

StatusChannel? _telegramChannel(List<StatusChannel>? channels) {
  if (channels == null) return null;
  for (final channel in channels) {
    if (channel.name.toLowerCase() == 'telegram') return channel;
  }
  return null;
}

/// Whether the product may show its "Connected" wording.
///
/// The single place that answers it, so no surface can invent a second
/// definition from a token, a username or an `enabled` flag.
bool telegramMayReportConnected(TelegramRuntimeState state) =>
    state == TelegramRuntimeState.running;
