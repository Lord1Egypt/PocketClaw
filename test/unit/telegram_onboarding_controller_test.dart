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
      deepLink:
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
  Harness({DateTime Function()? clock}) {
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
      storage: storage,
      clock: clock ?? DateTime.now,
    );
  }

  final client = FakeClient();
  final storage = MemoryStorage();
  final openedUrls = <String>[];
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

  test('backgrounding pauses polling and returning resumes it', () async {
    final h = Harness();
    await h.controller.start();
    expect(h.controller.isPolling, isTrue);

    h.controller.pausePolling();
    expect(h.controller.isPolling, isFalse);
    final callsWhilePaused = h.client.statusCalls;
    await Future<void>.delayed(const Duration(milliseconds: 60));
    expect(
      h.client.statusCalls,
      callsWhilePaused,
      reason: 'polling must stop while the app is backgrounded',
    );

    // The pairing itself must survive Telegram taking focus.
    expect(h.controller.pairing, isNotNull);
    expect(h.controller.stage, TelegramOnboardingStage.awaitingConfirmation);

    h.controller.resumePolling();
    expect(h.controller.isPolling, isTrue);
    await waitFor(() => h.client.statusCalls > callsWhilePaused);
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
      h.controller.pausePolling();
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
    h.controller.pausePolling();

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
