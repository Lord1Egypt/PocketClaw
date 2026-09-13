import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/telegram/telegram_config_writer.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_client.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_controller.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_models.dart';

/// A stand-in for PocketClaw's onboarding service. No test here touches a
/// network or a real Telegram bot.
class FakeClient extends TelegramOnboardingClient {
  FakeClient() : super(baseUrl: 'https://onboarding.invalid');

  int createCalls = 0;
  int statusCalls = 0;
  int collectCalls = 0;

  TelegramOnboardingException? createError;
  TelegramOnboardingException? statusError;
  TelegramOnboardingException? collectError;

  final List<TelegramPairingStatus> statusQueue = [];
  TelegramPairingStatus lastStatus = const TelegramPairingStatus(
    state: PairingState.pending,
  );

  DateTime expiresAt = DateTime.now().toUtc().add(const Duration(minutes: 10));

  /// Lets a test make the service answer with its own hosting URL, which is the
  /// shape PC-DEF-052 is about.
  String? deepLinkOverride;

  @override
  Future<TelegramPairing> createPairing() async {
    createCalls++;
    if (createError != null) throw createError!;
    return TelegramPairing(
      pairingId: 'pairing-$createCalls',
      pollToken: 'poll-$createCalls',
      suggestedUsername:
          'pocketclaw_abcd123$createCalls'
          '_bot',
      suggestedName: 'PocketClaw Agent',
      deepLink: deepLinkOverride ??
          'https://t.me/newbot/PocketClawSetupBot/'
              'pocketclaw_abcd123$createCalls'
              '_bot?name=PocketClaw%20Agent',
      qrPayload:
          'https://t.me/newbot/PocketClawSetupBot/'
          'pocketclaw_abcd123$createCalls'
          '_bot?name=PocketClaw%20Agent',
      expiresAt: expiresAt,
      pollInterval: const Duration(milliseconds: 10),
    );
  }

  @override
  Future<TelegramPairingStatus> fetchStatus(TelegramPairing pairing) async {
    statusCalls++;
    if (statusError != null) throw statusError!;
    if (statusQueue.isNotEmpty) lastStatus = statusQueue.removeAt(0);
    return lastStatus;
  }

  @override
  Future<TelegramBotCredentials> collectCredentials(
    TelegramPairing pairing,
  ) async {
    collectCalls++;
    if (collectError != null) throw collectError!;
    return const TelegramBotCredentials(
      token: '9001:CHILD-TOKEN',
      botUserId: 9001,
      botUsername: 'pocketclaw_abcd1231_bot',
      ownerUserId: 555,
    );
  }
}

class MemoryStorage implements PairingStorage {
  TelegramPairing? saved;
  int clearCalls = 0;

  @override
  Future<void> clear() async {
    clearCalls++;
    saved = null;
  }

  @override
  Future<TelegramPairing?> load() async => saved;

  @override
  Future<void> save(TelegramPairing pairing) async => saved = pairing;
}

class Harness {
  Harness({
    DateTime Function()? clock,
    Future<String?> Function(String)? resolveDeepLink,
    Future<bool> Function()? telegramRuntimeRunning,
    Duration? runtimeReadyTimeout,
  }) {
    if (resolveDeepLink != null) {
      resolvedLinks = resolveDeepLink;
    }
    if (telegramRuntimeRunning != null) {
      runtimeRunning = telegramRuntimeRunning;
    }
    controller = TelegramOnboardingController(
      client: client,
      configWriter: TelegramConfigWriter(
        writeCredentials: (credentials) async {
          if (configWriteFails) return false;
          savedCredentials = credentials;
          return true;
        },
      ),
      reloadCore: () async {
        reloads++;
        if (reloadFails) throw StateError('core did not restart');
      },
      openUrl: (url) async {
        openedUrls.add(url);
        return openSucceeds;
      },
      resolveDeepLink: (rawUrl) async {
        resolveRequests.add(rawUrl);
        return resolvedLinks(rawUrl);
      },
      telegramRuntimeRunning: () async {
        runtimeChecks++;
        return runtimeRunning();
      },
      // Short, so a readiness test does not sit out the production wait.
      runtimeReadyTimeout: runtimeReadyTimeout ?? const Duration(milliseconds: 400),
      runtimePollInterval: const Duration(milliseconds: 10),
      storage: storage,
      clock: clock ?? DateTime.now,
    );
  }

  final client = FakeClient();
  final storage = MemoryStorage();
  final openedUrls = <String>[];
  final resolveRequests = <String>[];

  /// Mirrors the production resolver's contract: a Telegram link passes through,
  /// anything else has to be resolved and may come back null.
  Future<String?> Function(String) resolvedLinks =
      (rawUrl) async => rawUrl.startsWith('https://t.me/') ? rawUrl : null;

  /// Core's report that the Telegram channel is running. Ready by default, so
  /// only a test about readiness has to think about it.
  Future<bool> Function() runtimeRunning = () async => true;
  int runtimeChecks = 0;

  late final TelegramOnboardingController controller;

  TelegramBotCredentials? savedCredentials;
  bool configWriteFails = false;
  bool reloadFails = false;
  bool openSucceeds = true;
  int reloads = 0;
}

/// Waits until [predicate] holds, letting the controller's timers run.
Future<void> waitFor(bool Function() predicate) async {
  for (var i = 0; i < 400; i++) {
    if (predicate()) return;
    await Future<void>.delayed(const Duration(milliseconds: 5));
  }
  fail('condition was never met');
}

void main() {
  _deepLinkGroup();
  _readinessGroup();
  _latencyGroup();
  test('start issues a pairing and begins waiting for Telegram', () async {
    final h = Harness();
    await h.controller.start();

    expect(h.controller.stage, TelegramOnboardingStage.awaitingConfirmation);
    expect(h.controller.pairing!.suggestedUsername, startsWith('pocketclaw_'));
    expect(
      h.controller.pairing!.deepLink,
      startsWith('https://t.me/newbot/PocketClawSetupBot/'),
    );
    expect(h.controller.isPolling, isTrue);
    expect(
      h.storage.saved,
      isNotNull,
      reason: 'the pairing must survive the app being killed',
    );
    h.controller.dispose();
  });

  test('the full happy path configures Core and reports connected', () async {
    final h = Harness();
    h.client.statusQueue.addAll(const [
      TelegramPairingStatus(state: PairingState.pending),
      TelegramPairingStatus(
        state: PairingState.created,
        botUsername: 'pocketclaw_abcd1231_bot',
      ),
      TelegramPairingStatus(
        state: PairingState.ready,
        botUsername: 'pocketclaw_abcd1231_bot',
      ),
    ]);

    await h.controller.start();
    await waitFor(
      () => h.controller.stage == TelegramOnboardingStage.connected,
    );

    expect(h.controller.connectedBotUsername, 'pocketclaw_abcd1231_bot');
    expect(
      h.controller.connectedChatUrl,
      'https://t.me/pocketclaw_abcd1231_bot',
    );
    expect(h.savedCredentials?.token, '9001:CHILD-TOKEN');
    expect(
      h.savedCredentials?.ownerUserId,
      555,
      reason: 'the creating user becomes the allow-list',
    );
    expect(h.reloads, 1, reason: 'Core must reload to pick up the channel');
    expect(
      h.client.collectCalls,
      1,
      reason: 'the token is collected exactly once',
    );
    expect(h.storage.saved, isNull, reason: 'the pairing is cleared when done');
    expect(h.controller.isPolling, isFalse);
    h.controller.dispose();
  });

  test('openTelegram opens the deep link and nothing secret', () async {
    final h = Harness();
    await h.controller.start();
    await h.controller.openTelegram();

    expect(h.openedUrls, hasLength(1));
    expect(h.openedUrls.single, h.controller.pairing!.deepLink);
    expect(
      h.openedUrls.single,
      isNot(contains(h.controller.pairing!.pollToken)),
    );
    expect(h.openedUrls.single, isNot(contains('CHILD-TOKEN')));
    h.controller.dispose();
  });

  test('a missing Telegram app is reported, not swallowed', () async {
    final h = Harness()..openSucceeds = false;
    await h.controller.start();
    await h.controller.openTelegram();

    expect(h.controller.stage, TelegramOnboardingStage.failed);
    expect(
      h.controller.errorKind,
      TelegramOnboardingErrorKind.telegramUnavailable,
    );
    h.controller.dispose();
  });

  // PC-DEF-056. Polling used to stop while the app was backgrounded, which is
  // exactly the window the user spends in Telegram. The pairing result could
  // not be consumed until they came back, so the bot chat Telegram showed them
  // was silent with no command menu.
  test('polling continues across the handoff to Telegram', () async {
    final h = Harness();
    await h.controller.start();
    expect(h.controller.isPolling, isTrue);

    final before = h.client.statusCalls;
    // The app is in Telegram here. Nothing stops the pairing being polled.
    await waitFor(() => h.client.statusCalls > before);

    expect(h.controller.pairing, isNotNull);
    expect(h.controller.stage, TelegramOnboardingStage.awaitingConfirmation);
    h.controller.dispose();
  });

  test('resuming is a no-op when polling is already live', () async {
    final h = Harness();
    await h.controller.start();
    expect(h.controller.isPolling, isTrue);

    h.controller.resumePolling();
    expect(h.controller.isPolling, isTrue);
    h.controller.dispose();
  });

  test(
    'resuming checks immediately rather than waiting a full interval',
    () async {
      final h = Harness();
      h.client.expiresAt = DateTime.now().toUtc().add(
        const Duration(minutes: 10),
      );
      await h.controller.start();
      final before = h.client.statusCalls;

      h.controller.resumePolling();
      await waitFor(() => h.client.statusCalls > before);
      h.controller.dispose();
    },
  );

  test('resuming does nothing once the flow has finished', () async {
    final h = Harness();
    h.client.statusQueue.add(
      const TelegramPairingStatus(state: PairingState.ready),
    );
    await h.controller.start();
    await waitFor(
      () => h.controller.stage == TelegramOnboardingStage.connected,
    );

    h.controller.resumePolling();
    expect(h.controller.isPolling, isFalse);
    h.controller.dispose();
  });

  test('a transient network error keeps the pairing alive', () async {
    final h = Harness();
    await h.controller.start();
    h.client.statusError = const TelegramOnboardingException(
      TelegramOnboardingErrorKind.network,
    );
    await waitFor(() => h.client.statusCalls > 2);

    expect(
      h.controller.stage,
      TelegramOnboardingStage.awaitingConfirmation,
      reason: 'a dropped poll must not end the pairing',
    );
    expect(h.controller.isPolling, isTrue);

    h.client.statusError = null;
    h.client.statusQueue.add(
      const TelegramPairingStatus(state: PairingState.ready),
    );
    await waitFor(
      () => h.controller.stage == TelegramOnboardingStage.connected,
    );
    h.controller.dispose();
  });

  test('an expired pairing is reported and cleared', () async {
    final h = Harness();
    h.client.expiresAt = DateTime.now().toUtc().subtract(
      const Duration(seconds: 1),
    );
    await h.controller.start();
    await waitFor(() => h.controller.stage == TelegramOnboardingStage.expired);

    expect(h.storage.saved, isNull);
    expect(h.controller.isPolling, isFalse);
    h.controller.dispose();
  });

  test(
    'a service-reported expiry is honoured even before the local deadline',
    () async {
      final h = Harness();
      h.client.statusQueue.add(
        const TelegramPairingStatus(state: PairingState.expired),
      );
      await h.controller.start();
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.expired,
      );
      h.controller.dispose();
    },
  );

  test('a failed pairing surfaces as a failure, not as success', () async {
    final h = Harness();
    h.client.statusQueue.add(
      const TelegramPairingStatus(
        state: PairingState.failed,
        reason: 'token_retrieval_failed',
      ),
    );
    await h.controller.start();
    await waitFor(() => h.controller.stage == TelegramOnboardingStage.failed);
    expect(h.controller.errorKind, TelegramOnboardingErrorKind.serviceError);
    h.controller.dispose();
  });

  test('a failed config write does not report connected', () async {
    final h = Harness()..configWriteFails = true;
    h.client.statusQueue.add(
      const TelegramPairingStatus(state: PairingState.ready),
    );
    await h.controller.start();
    await waitFor(() => h.controller.stage == TelegramOnboardingStage.failed);

    expect(
      h.controller.errorKind,
      TelegramOnboardingErrorKind.configurationFailed,
    );
    expect(h.controller.connectedBotUsername, isNull);
    h.controller.dispose();
  });

  test('a failed Core reload does not report connected', () async {
    final h = Harness()..reloadFails = true;
    h.client.statusQueue.add(
      const TelegramPairingStatus(state: PairingState.ready),
    );
    await h.controller.start();
    await waitFor(() => h.controller.stage == TelegramOnboardingStage.failed);

    expect(
      h.controller.errorKind,
      TelegramOnboardingErrorKind.configurationFailed,
    );
    expect(h.controller.connectedBotUsername, isNull);
    h.controller.dispose();
  });

  test('retry abandons the old pairing and issues a new one', () async {
    final h = Harness();
    await h.controller.start();
    final first = h.controller.pairing!.pairingId;

    await h.controller.retry();
    expect(h.controller.pairing!.pairingId, isNot(first));
    expect(h.client.createCalls, 2);
    expect(h.controller.stage, TelegramOnboardingStage.awaitingConfirmation);
    h.controller.dispose();
  });

  test('a rate-limited service is reported distinctly', () async {
    final h = Harness();
    h.client.createError = const TelegramOnboardingException(
      TelegramOnboardingErrorKind.rateLimited,
    );
    await h.controller.start();

    expect(h.controller.stage, TelegramOnboardingStage.failed);
    expect(h.controller.errorKind, TelegramOnboardingErrorKind.rateLimited);
    h.controller.dispose();
  });

  test('restore picks up a pairing left by a killed app', () async {
    final h = Harness();
    h.storage.saved = TelegramPairing(
      pairingId: 'restored',
      pollToken: 'poll-restored',
      suggestedUsername: 'pocketclaw_restored_bot',
      suggestedName: 'PocketClaw Agent',
      deepLink:
          'https://t.me/newbot/PocketClawSetupBot/pocketclaw_restored_bot',
      qrPayload:
          'https://t.me/newbot/PocketClawSetupBot/pocketclaw_restored_bot',
      expiresAt: DateTime.now().toUtc().add(const Duration(minutes: 5)),
      pollInterval: const Duration(milliseconds: 10),
    );

    await h.controller.restore();
    expect(h.controller.stage, TelegramOnboardingStage.awaitingConfirmation);
    expect(h.controller.pairing!.pairingId, 'restored');
    expect(h.controller.isPolling, isTrue);
    expect(
      h.client.createCalls,
      0,
      reason: 'restore must not start a new pairing',
    );
    h.controller.dispose();
  });

  test(
    'restore discards a pairing that expired while the app was gone',
    () async {
      final h = Harness();
      h.storage.saved = TelegramPairing(
        pairingId: 'stale',
        pollToken: 'poll-stale',
        suggestedUsername: 'pocketclaw_stale000_bot',
        suggestedName: 'PocketClaw Agent',
        deepLink:
            'https://t.me/newbot/PocketClawSetupBot/pocketclaw_stale000_bot',
        qrPayload:
            'https://t.me/newbot/PocketClawSetupBot/pocketclaw_stale000_bot',
        expiresAt: DateTime.now().toUtc().subtract(const Duration(minutes: 1)),
        pollInterval: const Duration(milliseconds: 10),
      );

      await h.controller.restore();
      expect(h.controller.stage, TelegramOnboardingStage.idle);
      expect(h.storage.clearCalls, greaterThan(0));
      h.controller.dispose();
    },
  );

  test('timeRemaining counts down and never goes negative', () async {
    var now = DateTime.utc(2026, 8, 26, 12, 0, 0);
    final h = Harness(clock: () => now);
    h.client.expiresAt = now.add(const Duration(minutes: 10));
    await h.controller.start();

    expect(h.controller.timeRemaining, const Duration(minutes: 10));
    now = now.add(const Duration(minutes: 4));
    expect(h.controller.timeRemaining, const Duration(minutes: 6));
    now = now.add(const Duration(minutes: 30));
    expect(h.controller.timeRemaining, Duration.zero);
    h.controller.dispose();
  });

  test('reset returns to the beginning and stops polling', () async {
    final h = Harness();
    await h.controller.start();
    await h.controller.reset();

    expect(h.controller.stage, TelegramOnboardingStage.idle);
    expect(h.controller.pairing, isNull);
    expect(h.controller.isPolling, isFalse);
    expect(h.storage.saved, isNull);
    h.controller.dispose();
  });

  test('an unconfigured build reports that, and never calls out', () async {
    final h = Harness();
    final controller = TelegramOnboardingController(
      client: h.client,
      configWriter: TelegramConfigWriter(writeCredentials: (_) async => true),
      reloadCore: () async {},
      openUrl: (_) async => true,
      serviceConfigured: false,
    );

    await controller.start();
    expect(controller.stage, TelegramOnboardingStage.failed);
    expect(controller.errorKind, TelegramOnboardingErrorKind.notConfigured);
    expect(
      h.client.createCalls,
      0,
      reason: 'no request may be made without a configured endpoint',
    );
    controller.dispose();
    h.controller.dispose();
  });

  test('an unknown state from a newer service is treated as failure', () {
    expect(PairingState.parse('something_new'), PairingState.failed);
    expect(PairingState.parse('ready'), PairingState.ready);
  });
}

/// PC-DEF-052. The user must see Telegram, never the host that issued the link.
void _deepLinkGroup() {
  group('PC-DEF-052 direct Telegram launch', () {
    test('opens the resolved Telegram link, not the service link', () async {
      const hosted = 'https://pocketclaw-telegram-setup.vercel.app/go/abc';
      const telegram = 'https://t.me/newbot/Mgr/pocketclaw_bot';

      final h = Harness();
      h.client.deepLinkOverride = hosted;
      h.resolvedLinks = (raw) async => raw == hosted ? telegram : null;

      await h.controller.start();
      await h.controller.openTelegram();

      expect(h.resolveRequests, [hosted],
          reason: 'the service link is resolved in the background');
      expect(h.openedUrls, [telegram]);
      expect(h.openedUrls.single, isNot(contains('vercel.app')),
          reason: 'the hosting origin must never be user-visible navigation');
      expect(h.controller.stage, TelegramOnboardingStage.awaitingConfirmation);
    });

    test('refuses to open a link that does not resolve to Telegram', () async {
      const hosted = 'https://pocketclaw-telegram-setup.vercel.app/go/abc';

      final h = Harness();
      h.client.deepLinkOverride = hosted;
      h.resolvedLinks = (_) async => null;

      await h.controller.start();
      await h.controller.openTelegram();

      expect(h.openedUrls, isEmpty,
          reason: 'opening the hosting page is the defect, not the fallback');
      expect(h.controller.stage, TelegramOnboardingStage.failed);
      expect(
        h.controller.errorKind,
        TelegramOnboardingErrorKind.telegramLinkUnavailable,
      );
    });

    test('a Telegram link is opened as-is', () async {
      final h = Harness();
      await h.controller.start();
      await h.controller.openTelegram();

      expect(h.openedUrls.single, startsWith('https://t.me/'));
    });

    // Telegram missing is a different failure from no link to give it: one tells
    // the user to install Telegram, the other to try setup again.
    test('a failed launch of a valid link reports Telegram unavailable',
        () async {
      final h = Harness()..openSucceeds = false;
      await h.controller.start();
      await h.controller.openTelegram();

      expect(
        h.controller.errorKind,
        TelegramOnboardingErrorKind.telegramUnavailable,
      );
    });

    test('no bot token or poll token ever appears in an opened URI', () async {
      final h = Harness();
      await h.controller.start();
      await h.controller.openTelegram();
      await h.controller.openBotChat();

      for (final url in h.openedUrls) {
        expect(url, isNot(contains(h.controller.pairing!.pollToken)));
        // A Telegram bot token is `<digits>:<base64url>`; nothing shaped like a
        // credential belongs in a URL the OS is handed.
        expect(RegExp(r'\d{6,}:[A-Za-z0-9_-]{20,}').hasMatch(url), isFalse,
            reason: url);
      }
    });

    test('a reconnect resolves and launches again for the new pairing',
        () async {
      final h = Harness();
      await h.controller.start();
      await h.controller.openTelegram();

      h.controller.reset();
      await h.controller.start();
      await h.controller.openTelegram();

      expect(h.resolveRequests.length, 2,
          reason: 'each pairing gets its own resolution');
      expect(h.openedUrls.length, 2);
      expect(h.openedUrls[0], isNot(h.openedUrls[1]),
          reason: 'a reconnect pairs a different suggested bot');
    });
  });
}

/// PC-DEF-056. The user must never be dropped into a bot chat that is not ready.
///
/// The physical symptom: onboarding finished, Telegram opened the new bot chat,
/// and the bot was silent with no command menu until the owner returned to
/// PocketClaw and pressed Open Chat a second time. Two causes, both here.
void _readinessGroup() {
  group('PC-DEF-056 runtime readiness', () {
    /// Drives a pairing to the point where the service says it is ready.
    Future<Harness> completing({
      Future<bool> Function()? runtime,
      Duration? timeout,
    }) async {
      final h = Harness(
        telegramRuntimeRunning: runtime,
        runtimeReadyTimeout: timeout,
      );
      h.client.statusQueue.add(
        const TelegramPairingStatus(state: PairingState.ready),
      );
      await h.controller.start();
      return h;
    }

    test('connected is reached only once Core reports the channel running',
        () async {
      var running = false;
      final h = await completing(
        runtime: () async => running,
        timeout: const Duration(seconds: 5),
      );

      // Configuration is written and Core restarted, but the channel is not up.
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.startingRuntime,
      );
      expect(h.savedCredentials, isNotNull,
          reason: 'the token is persisted before the wait, not after');
      expect(h.controller.stage, isNot(TelegramOnboardingStage.connected));

      running = true;
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.connected,
      );
      h.controller.dispose();
    });

    // The whole point: no second action by the user.
    test('the bot chat is opened automatically on reaching connected', () async {
      final h = await completing();
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.connected,
      );

      expect(
        h.openedUrls.where((u) => u.startsWith('https://t.me/')).length,
        greaterThan(0),
        reason: 'reaching the bot must take no second Open Chat press',
      );
      h.controller.dispose();
    });

    test('the bot chat is never opened before the runtime is running', () async {
      var running = false;
      final h = await completing(
        runtime: () async => running,
        timeout: const Duration(seconds: 5),
      );
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.startingRuntime,
      );

      expect(h.openedUrls, isEmpty,
          reason: 'an inactive bot chat is the defect');

      running = true;
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.connected,
      );
      expect(h.openedUrls, isNotEmpty);
      h.controller.dispose();
    });

    test('the automatic open happens exactly once', () async {
      final h = await completing();
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.connected,
      );
      final afterConnect = h.openedUrls.length;

      // A resume, a rebuild and a second listener callback must not reopen it.
      h.controller.resumePolling();
      h.controller.resumePolling();
      await Future<void>.delayed(const Duration(milliseconds: 40));

      expect(h.openedUrls.length, afterConnect);
      h.controller.dispose();
    });

    // Bounded, because an unbounded wait is a hang.
    test('a runtime that never starts is reported, not waited on forever',
        () async {
      final h = await completing(
        runtime: () async => false,
        timeout: const Duration(milliseconds: 150),
      );

      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.failed,
      );
      expect(h.controller.errorKind, TelegramOnboardingErrorKind.runtimeNotReady);
      expect(h.openedUrls, isEmpty,
          reason: 'a failed start must not send the user into a dead chat');
      // The configuration is sound; only the start is outstanding.
      expect(h.savedCredentials, isNotNull);
      h.controller.dispose();
    });

    // Silence right after a restart is not a failure: Core is not reporting yet.
    test('a status read that throws is treated as not-yet-running', () async {
      var attempts = 0;
      final h = await completing(
        runtime: () async {
          attempts++;
          if (attempts < 3) throw StateError('host unavailable');
          return true;
        },
        timeout: const Duration(seconds: 5),
      );

      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.connected,
      );
      expect(attempts, greaterThanOrEqualTo(3));
      h.controller.dispose();
    });

    test('the configuration is applied exactly once across the wait', () async {
      var running = false;
      final h = await completing(
        runtime: () async => running,
        timeout: const Duration(seconds: 5),
      );
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.startingRuntime,
      );
      final reloadsDuringWait = h.reloads;

      running = true;
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.connected,
      );

      expect(h.reloads, reloadsDuringWait,
          reason: 'waiting for readiness must not re-apply the configuration');
      expect(h.reloads, 1);
      expect(h.client.collectCalls, 1,
          reason: 'the token is collected exactly once');
      h.controller.dispose();
    });

    test('a reconnect runs the readiness gate again for the new bot', () async {
      final h = await completing();
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.connected,
      );
      final firstOpens = h.openedUrls.length;

      h.client.statusQueue.add(
        const TelegramPairingStatus(state: PairingState.ready),
      );
      await h.controller.retry();
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.connected,
      );

      expect(h.openedUrls.length, greaterThan(firstOpens),
          reason: 'a replacement bot gets its own automatic open');
      expect(h.client.collectCalls, 2);
      h.controller.dispose();
    });

    test('cancelling before readiness opens nothing', () async {
      var running = false;
      final h = await completing(
        runtime: () async => running,
        timeout: const Duration(seconds: 5),
      );
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.startingRuntime,
      );

      await h.controller.reset();
      running = true;
      await Future<void>.delayed(const Duration(milliseconds: 60));

      expect(h.controller.stage, TelegramOnboardingStage.idle);
      expect(h.openedUrls, isEmpty);
      h.controller.dispose();
    });

    // The owner contract from PC-DEF-044: exactly one paired numeric owner.
    test('the owner identity written to Core is unchanged by the gate', () async {
      final h = await completing();
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.connected,
      );

      final credentials = h.savedCredentials!;
      expect(credentials.ownerUserId, greaterThan(0));
      expect(credentials.token, isNotEmpty);
      h.controller.dispose();
    });

    // No sleep-based patch: the gate is driven by the readiness signal, so a
    // runtime that comes up immediately must not be made to wait.
    test('a runtime already running does not delay the flow', () async {
      final h = await completing();
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.connected,
      );

      expect(h.runtimeChecks, lessThanOrEqualTo(2),
          reason: 'readiness is polled, not slept through');
      h.controller.dispose();
    });
  });
}

/// The owner's 15-25 s observation, instrumented rather than optimised.
void _latencyGroup() {
  group('onboarding latency marks', () {
    test('records how long the runtime wait took', () async {
      final h = Harness();
      h.client.statusQueue.add(
        const TelegramPairingStatus(state: PairingState.ready),
      );
      await h.controller.start();
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.connected,
      );

      expect(h.controller.runtimeReadyLatency, isNotNull,
          reason: 'the stage has to be attributable, not guessed at');
      expect(h.controller.onboardingLatency, isNotNull);
      expect(h.controller.runtimeReadyLatency!.inMilliseconds,
          greaterThanOrEqualTo(0));
      h.controller.dispose();
    });

    test('clears the marks when a new flow begins', () async {
      final h = Harness();
      h.client.statusQueue.add(
        const TelegramPairingStatus(state: PairingState.ready),
      );
      await h.controller.start();
      await waitFor(
        () => h.controller.stage == TelegramOnboardingStage.connected,
      );
      expect(h.controller.runtimeReadyLatency, isNotNull);

      await h.controller.retry();
      // A fresh flow must not report the previous flow's timing.
      expect(h.controller.runtimeReadyLatency, isNull);
      h.controller.dispose();
    });
  });
}
