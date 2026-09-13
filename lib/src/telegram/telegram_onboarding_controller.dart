import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';

import '../core/pocketclaw_channel.dart';
import '../core/status_snapshot.dart';
import '../core/telegram_runtime_state.dart';
import 'telegram_config_writer.dart';
import 'telegram_deep_link.dart';
import 'telegram_onboarding_client.dart';
import 'telegram_onboarding_models.dart';

/// What the onboarding screen is currently showing.
enum TelegramOnboardingStage {
  /// Nothing started yet.
  idle,

  /// Asking the service for a pairing.
  creatingPairing,

  /// The link and QR are ready; waiting for the user to confirm in Telegram.
  awaitingConfirmation,

  /// Telegram reported the bot; its token is being retrieved.
  botCreated,

  /// Writing the token into Core and reloading the Telegram channel.
  configuring,

  /// The configuration is saved and Core is restarting; waiting for Core to
  /// report the Telegram channel as running.
  ///
  /// PC-DEF-056. Configuration applied is not the same fact as runtime running,
  /// and the flow used to declare success on the first. The user was then handed
  /// a bot chat whose channel had not started, so the bot did not answer and had
  /// no command menu.
  startingRuntime,

  /// Done: Core reports the Telegram channel running, so the bot can answer.
  connected,

  /// The pairing window closed before Telegram confirmed anything.
  expired,

  /// Something went wrong; see [TelegramOnboardingController.errorKind].
  failed,
}

/// Persists a pairing across an app restart.
///
/// Android can kill PocketClaw while the user is confirming in Telegram. Only
/// the pairing's own identifiers are stored, in app-private storage: they are
/// short-lived, scoped to one pairing, and grant nothing else. The bot token is
/// never stored here — it goes straight into Core's configuration.
abstract class PairingStorage {
  Future<void> save(TelegramPairing pairing);
  Future<TelegramPairing?> load();
  Future<void> clear();
}

/// Drives the Telegram managed-bot onboarding flow.
class TelegramOnboardingController extends ChangeNotifier {
  TelegramOnboardingController({
    required TelegramOnboardingClient client,
    required TelegramConfigWriter configWriter,
    required Future<void> Function() reloadCore,
    required Future<bool> Function(String url) openUrl,
    Future<String?> Function(String rawUrl)? resolveDeepLink,
    Future<bool> Function()? telegramRuntimeRunning,
    Duration? runtimeReadyTimeout,
    Duration? runtimePollInterval,
    PairingStorage? storage,
    DateTime Function()? clock,
    bool serviceConfigured = true,
  })  : _client = client,
        _configWriter = configWriter,
        _reloadCore = reloadCore,
        _openUrl = openUrl,
        _resolveDeepLink = resolveDeepLink ?? _defaultResolveDeepLink,
        _telegramRuntimeRunning =
            telegramRuntimeRunning ?? defaultTelegramRuntimeRunning,
        _runtimeReadyTimeout = runtimeReadyTimeout ?? const Duration(seconds: 45),
        _runtimePollInterval = runtimePollInterval ?? const Duration(seconds: 1),
        _storage = storage,
        _clock = clock ?? DateTime.now,
        _serviceConfigured = serviceConfigured;

  final TelegramOnboardingClient _client;
  final TelegramConfigWriter _configWriter;
  final Future<void> Function() _reloadCore;
  final Future<bool> Function(String url) _openUrl;
  final Future<String?> Function(String rawUrl) _resolveDeepLink;

  /// Whether Core reports the Telegram channel as running. The single
  /// authoritative readiness signal; see PC-DEF-027's resolver.
  final Future<bool> Function() _telegramRuntimeRunning;

  /// Bounded, because an unbounded wait is a hang. On expiry the flow reports
  /// that the configuration is saved but the runtime has not started.
  final Duration _runtimeReadyTimeout;
  final Duration _runtimePollInterval;

  final PairingStorage? _storage;

  /// Whether this pairing has already had its bot chat opened automatically.
  bool _autoOpenedBotChat = false;

  /// When the current flow began, for the latency marks below.
  DateTime? _startedAt;

  /// How long Core took to report the Telegram channel running, once known.
  ///
  /// Recorded so the owner's 15-25 s observation can be attributed to a stage
  /// instead of guessed at. Null until the flow reaches connected.
  Duration? runtimeReadyLatency;

  /// How long the whole flow took, from start to connected.
  Duration? onboardingLatency;
  final DateTime Function() _clock;

  /// False when this build has no PocketClaw onboarding endpoint. The flow
  /// then reports that automatic setup is unavailable instead of attempting a
  /// request against an unset URL and calling the result a network problem.
  final bool _serviceConfigured;

  TelegramOnboardingStage _stage = TelegramOnboardingStage.idle;
  TelegramPairing? _pairing;
  TelegramOnboardingErrorKind? _errorKind;
  String? _connectedBotUsername;
  Timer? _pollTimer;
  bool _disposed = false;

  TelegramOnboardingStage get stage => _stage;
  TelegramPairing? get pairing => _pairing;
  TelegramOnboardingErrorKind? get errorKind => _errorKind;

  /// The paired bot's `@username`, once connected. Never the token.
  String? get connectedBotUsername => _connectedBotUsername;

  /// The chat link for the Open Chat action.
  String? get connectedChatUrl =>
      _connectedBotUsername == null ? null : 'https://t.me/$_connectedBotUsername';

  /// Whether polling is currently running. Exposed for tests and diagnostics.
  bool get isPolling => _pollTimer?.isActive ?? false;

  /// How long the current pairing has left, or null when there is none.
  Duration? get timeRemaining {
    final pairing = _pairing;
    if (pairing == null) return null;
    final remaining = pairing.expiresAt.difference(_clock().toUtc());
    return remaining.isNegative ? Duration.zero : remaining;
  }

  /// Restores a pairing left behind by a previous run of the app, so a process
  /// death while the user was in Telegram does not lose the session.
  Future<void> restore() async {
    final storage = _storage;
    if (storage == null || _stage != TelegramOnboardingStage.idle) return;

    final saved = await storage.load();
    if (saved == null) return;
    if (saved.isExpiredAt(_clock().toUtc())) {
      await storage.clear();
      return;
    }
    _pairing = saved;
    _setStage(TelegramOnboardingStage.awaitingConfirmation);
    _startPolling();
  }

  /// Begins a new pairing.
  Future<void> start() async {
    _stopPolling();
    _errorKind = null;
    _connectedBotUsername = null;
    _startedAt = _clock();
    runtimeReadyLatency = null;
    onboardingLatency = null;

    if (!_serviceConfigured) {
      _fail(TelegramOnboardingErrorKind.notConfigured);
      return;
    }
    _setStage(TelegramOnboardingStage.creatingPairing);

    try {
      final pairing = await _client.createPairing();
      if (_disposed) return;
      _pairing = pairing;
      await _storage?.save(pairing);
      _setStage(TelegramOnboardingStage.awaitingConfirmation);
      _startPolling();
    } on TelegramOnboardingException catch (error) {
      _fail(error.kind);
    }
  }

  /// Opens the managed-bot creation screen in Telegram.
  ///
  /// The link carries no secret: it names the manager bot, the suggested
  /// username, and the suggested display name, all of which Telegram is about
  /// to show the user anyway.
  ///
  /// PC-DEF-052. The service's link is resolved to a Telegram destination first,
  /// in the background, and only Telegram is ever opened. A deployment that
  /// serves the setup link from its own redirect endpoint used to put that
  /// hosting origin on screen for a moment on the way through. A link that does
  /// not resolve to Telegram is refused, not opened: opening it is the defect,
  /// and a URL taken from a network response is untrusted input being handed
  /// straight to the OS.
  Future<void> openTelegram() async {
    final pairing = _pairing;
    if (pairing == null) return;

    final target = await _resolveDeepLink(pairing.deepLink);
    if (target == null) {
      _fail(TelegramOnboardingErrorKind.telegramLinkUnavailable);
      return;
    }

    final opened = await _openUrl(target);
    if (!opened) {
      _fail(TelegramOnboardingErrorKind.telegramUnavailable);
    }
  }

  /// Opens a chat with the newly paired bot.
  Future<bool> openBotChat() async {
    final url = connectedChatUrl;
    if (url == null) return false;
    return _openUrl(url);
  }

  /// Checks once immediately and makes sure polling is running.
  ///
  /// PC-DEF-056. Polling used to be *stopped* while the app was backgrounded,
  /// which is precisely the window the user spends in Telegram confirming the
  /// bot. The pairing result therefore could not be consumed until they came
  /// back: the token was not collected, Core had no Telegram channel, and the
  /// bot chat Telegram dropped them into was silent with no command menu.
  ///
  /// Polling now continues across the handoff, so the result is consumed at the
  /// earliest moment it exists. It is still bounded — the pairing's own
  /// `expiresAt` ends it — and if Android kills the process instead, `restore`
  /// picks the pairing back up. This is called on resume as a catch-up for that
  /// case and is a no-op when polling is already live.
  void resumePolling() {
    if (_stage != TelegramOnboardingStage.awaitingConfirmation &&
        _stage != TelegramOnboardingStage.botCreated) {
      return;
    }
    if (isPolling) return;
    _startPolling();
    unawaited(_poll());
  }

  /// Abandons the current pairing and starts a fresh one.
  Future<void> retry() async {
    await _storage?.clear();
    _pairing = null;
    _autoOpenedBotChat = false;
    await start();
  }

  /// Returns to the initial state, discarding any in-flight pairing.
  Future<void> reset() async {
    _stopPolling();
    await _storage?.clear();
    _pairing = null;
    _errorKind = null;
    _connectedBotUsername = null;
    _autoOpenedBotChat = false;
    _setStage(TelegramOnboardingStage.idle);
  }

  void _startPolling() {
    _stopPolling();
    final interval = _pairing?.pollInterval ?? const Duration(seconds: 2);
    _pollTimer = Timer.periodic(interval, (_) => unawaited(_poll()));
  }

  void _stopPolling() {
    _pollTimer?.cancel();
    _pollTimer = null;
  }

  Future<void> _poll() async {
    final pairing = _pairing;
    if (pairing == null || _disposed) return;
    if (_stage != TelegramOnboardingStage.awaitingConfirmation &&
        _stage != TelegramOnboardingStage.botCreated) {
      return;
    }

    if (pairing.isExpiredAt(_clock().toUtc())) {
      await _expire();
      return;
    }

    final TelegramPairingStatus status;
    try {
      status = await _client.fetchStatus(pairing);
    } on TelegramOnboardingException catch (error) {
      // A transient network failure must not end the pairing: the user may be
      // mid-handoff to Telegram with the radio busy. Keep polling until the
      // pairing's own deadline decides.
      if (error.kind == TelegramOnboardingErrorKind.network) return;
      _fail(error.kind);
      return;
    }
    if (_disposed) return;

    switch (status.state) {
      case PairingState.pending:
        return;
      case PairingState.created:
        _setStage(TelegramOnboardingStage.botCreated);
        return;
      case PairingState.ready:
        _stopPolling();
        await _completePairing(pairing);
        return;
      case PairingState.expired:
        await _expire();
        return;
      case PairingState.failed:
        _stopPolling();
        _fail(TelegramOnboardingErrorKind.serviceError);
        return;
    }
  }

  Future<void> _completePairing(TelegramPairing pairing) async {
    _setStage(TelegramOnboardingStage.configuring);
    try {
      final credentials = await _client.collectCredentials(pairing);
      if (_disposed) return;

      await _configWriter.apply(credentials);
      if (_disposed) return;

      // Core reads channel configuration at startup, so the new bot only comes
      // online after a reload.
      await _reloadCore();
      if (_disposed) return;

      _connectedBotUsername = credentials.botUsername;
      await _storage?.clear();

      // PC-DEF-056. A reload that returns has been *requested*, not completed:
      // restartCore hands Core one intent and comes back. Declaring success here
      // handed the user a bot whose channel had not started, so it did not answer
      // and had no command menu until something later brought it up.
      //
      // Core registers the Telegram command menu as part of starting the
      // channel, so the channel reporting running is the earliest point at which
      // the bot can answer. The app cannot observe setMyCommands itself -- Core
      // exposes no per-channel command state -- so running is the authoritative
      // signal available, and it is the one the gate uses.
      _setStage(TelegramOnboardingStage.startingRuntime);
      // The owner measured 15-25 s to the first Telegram reply on a fresh
      // onboarding and asked for the stages to be instrumented rather than
      // optimised by guesswork. These are wall-clock marks, not a fix: they say
      // which step the wait is actually in.
      final runtimeWaitStarted = _clock();
      if (!await _awaitTelegramRunning()) {
        if (_disposed) return;
        // A wait the user cancelled is not a runtime failure, and must not
        // overwrite the state they moved to.
        if (_stage != TelegramOnboardingStage.startingRuntime) return;
        // The token and owner are persisted and sound; only the start is
        // outstanding. Never "connected", and never an open bot chat.
        _fail(TelegramOnboardingErrorKind.runtimeNotReady);
        return;
      }
      if (_disposed || _stage != TelegramOnboardingStage.startingRuntime) return;

      runtimeReadyLatency = _clock().difference(runtimeWaitStarted);
      final startedAt = _startedAt;
      if (startedAt != null) {
        onboardingLatency = _clock().difference(startedAt);
      }
      debugPrint(
        'pocketclaw.onboarding stage=telegram_running '
        'runtime_wait_ms=${runtimeReadyLatency!.inMilliseconds} '
        'onboarding_total_ms=${onboardingLatency?.inMilliseconds ?? -1}',
      );

      _setStage(TelegramOnboardingStage.connected);
      await _openConnectedBotChatOnce();
    } on TelegramOnboardingException catch (error) {
      _fail(error.kind);
    } catch (_) {
      // A reload failure lands here. The token is already written, so the
      // configuration is sound even though the channel has not come up; the UI
      // says so and offers a retry rather than claiming success.
      _fail(TelegramOnboardingErrorKind.configurationFailed);
    }
  }

  /// Polls the readiness signal until it says running, or the wait expires.
  ///
  /// Silence is not failure: right after a restart Core is not reporting at all,
  /// and PC-DEF-027 is explicit that an absent runtime report is silence rather
  /// than a verdict. So a false answer keeps waiting and only the deadline
  /// decides -- which is what makes this bounded rather than a hang.
  Future<bool> _awaitTelegramRunning() async {
    final deadline = _clock().add(_runtimeReadyTimeout);
    // The stage is re-checked every iteration, not just _disposed: a reset or a
    // retry while the wait is in flight has to abandon it. Without that, a
    // cancelled pairing whose runtime came up later went on to declare itself
    // connected and open a bot chat the user had walked away from.
    while (!_disposed && _stage == TelegramOnboardingStage.startingRuntime) {
      bool running;
      try {
        running = await _telegramRuntimeRunning();
      } catch (_) {
        // A failed status read is silence too.
        running = false;
      }
      if (_disposed || _stage != TelegramOnboardingStage.startingRuntime) {
        return false;
      }
      if (running) return true;
      if (!_clock().isBefore(deadline)) return false;
      await Future<void>.delayed(_runtimePollInterval);
    }
    return false;
  }

  /// Opens the finished bot's chat, at most once per completed pairing.
  ///
  /// The contract is that reaching the bot takes no second action by the user, so
  /// the flow opens it rather than waiting to be asked. Exactly once: a rebuild,
  /// a resume or a second listener callback must not reopen Telegram.
  ///
  /// A failure to open is deliberately not an error. The configuration is live
  /// and the bot works; the screen still offers Open Chat.
  Future<void> _openConnectedBotChatOnce() async {
    if (_autoOpenedBotChat) return;
    _autoOpenedBotChat = true;
    await openBotChat();
  }

  Future<void> _expire() async {
    _stopPolling();
    await _storage?.clear();
    _setStage(TelegramOnboardingStage.expired);
  }

  void _fail(TelegramOnboardingErrorKind kind) {
    _stopPolling();
    _errorKind = kind;
    _setStage(TelegramOnboardingStage.failed);
  }

  void _setStage(TelegramOnboardingStage stage) {
    if (_disposed) return;
    _stage = stage;
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    _stopPolling();
    super.dispose();
  }
}

/// A [PairingStorage] backed by `shared_preferences`.
class SharedPreferencesPairingStorage implements PairingStorage {
  const SharedPreferencesPairingStorage(this._read, this._write, this._remove);

  /// Injected accessors keep this class free of a direct plugin dependency,
  /// which is what lets the controller be tested without a platform channel.
  final Future<String?> Function(String key) _read;
  final Future<void> Function(String key, String value) _write;
  final Future<void> Function(String key) _remove;

  static const String storageKey = 'pocketclaw.telegram.pairing';

  @override
  Future<void> save(TelegramPairing pairing) =>
      _write(storageKey, jsonEncode(pairing.toStorageJson()));

  @override
  Future<TelegramPairing?> load() async {
    final raw = await _read(storageKey);
    if (raw == null || raw.isEmpty) return null;
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map<String, dynamic>) return null;
      return TelegramPairing.fromStorageJson(decoded);
    } catch (_) {
      return null;
    }
  }

  @override
  Future<void> clear() => _remove(storageKey);
}

/// The production deep-link resolver.
///
/// A short-lived client per resolution: onboarding resolves one link, once, and
/// a resolver held for the life of the controller would keep a connection open
/// across the whole flow for no benefit.
Future<String?> _defaultResolveDeepLink(String rawUrl) async {
  final resolver = TelegramDeepLinkResolver();
  try {
    return await resolver.resolve(rawUrl);
  } finally {
    resolver.close();
  }
}

/// The production readiness signal: Core's own report that the Telegram channel
/// is running.
///
/// Routed through [resolveTelegramRuntimeState] so there is one definition of
/// "Telegram is running" in the app (PC-DEF-027) rather than a second one
/// invented here from a token or a username. `configuredAndValid` is true
/// because this is only ever asked immediately after the flow wrote the
/// configuration itself.
Future<bool> defaultTelegramRuntimeRunning() async {
  try {
    final health = await PocketClawChannel.checkHealth(detail: true);
    if (health['isHealthy'] != true) return false;
    final snapshot = StatusSnapshot.tryParse(health['detail'] as String?);
    final state = resolveTelegramRuntimeState(
      configuredAndValid: true,
      channels: snapshot?.channels,
    );
    return telegramMayReportConnected(state);
  } catch (_) {
    // An unavailable host or a malformed snapshot is silence, not a verdict.
    return false;
  }
}
