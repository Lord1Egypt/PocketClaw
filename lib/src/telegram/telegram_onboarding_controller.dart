import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';

import 'telegram_config_writer.dart';
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

  /// Done. The bot is configured and ready to chat.
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
    PairingStorage? storage,
    DateTime Function()? clock,
    bool serviceConfigured = true,
  })  : _client = client,
        _configWriter = configWriter,
        _reloadCore = reloadCore,
        _openUrl = openUrl,
        _storage = storage,
        _clock = clock ?? DateTime.now,
        _serviceConfigured = serviceConfigured;

  final TelegramOnboardingClient _client;
  final TelegramConfigWriter _configWriter;
  final Future<void> Function() _reloadCore;
  final Future<bool> Function(String url) _openUrl;
  final PairingStorage? _storage;
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
  Future<void> openTelegram() async {
    final pairing = _pairing;
    if (pairing == null) return;
    final opened = await _openUrl(pairing.deepLink);
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

  /// Suspends polling while the app is in the background. The pairing itself
  /// is untouched, so returning from Telegram resumes the same session.
  void pausePolling() => _stopPolling();

  /// Resumes polling and checks once immediately, so returning from Telegram
  /// does not wait out a full poll interval before showing progress.
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
    await start();
  }

  /// Returns to the initial state, discarding any in-flight pairing.
  Future<void> reset() async {
    _stopPolling();
    await _storage?.clear();
    _pairing = null;
    _errorKind = null;
    _connectedBotUsername = null;
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
      _setStage(TelegramOnboardingStage.connected);
    } on TelegramOnboardingException catch (error) {
      _fail(error.kind);
    } catch (_) {
      // A reload failure lands here. The token is already written, so the
      // configuration is sound even though the channel has not come up; the UI
      // says so and offers a retry rather than claiming success.
      _fail(TelegramOnboardingErrorKind.configurationFailed);
    }
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
