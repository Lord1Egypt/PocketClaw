import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/telegram/telegram_config_writer.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_client.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_controller.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_models.dart';
import 'package:pocketclaw/src/ui/telegram_onboarding_page.dart';

class StubClient extends TelegramOnboardingClient {
  StubClient() : super(baseUrl: 'https://onboarding.invalid');

  final statusQueue = <TelegramPairingStatus>[];
  TelegramPairingStatus last = const TelegramPairingStatus(
    state: PairingState.pending,
  );
  TelegramOnboardingException? createError;

  @override
  Future<TelegramPairing> createPairing() async {
    if (createError != null) throw createError!;
    return TelegramPairing(
      pairingId: 'pairing-1',
      pollToken: 'poll-secret-value',
      suggestedUsername: 'pocketclaw_abcd1234_bot',
      suggestedName: 'PocketClaw Agent',
      deepLink:
          'https://t.me/newbot/PocketClawSetupBot/'
          'pocketclaw_abcd1234_bot?name=PocketClaw%20Agent',
      qrPayload:
          'https://t.me/newbot/PocketClawSetupBot/'
          'pocketclaw_abcd1234_bot?name=PocketClaw%20Agent',
      expiresAt: DateTime.now().toUtc().add(const Duration(minutes: 10)),
      pollInterval: const Duration(milliseconds: 20),
    );
  }

  @override
  Future<TelegramPairingStatus> fetchStatus(TelegramPairing pairing) async {
    if (statusQueue.isNotEmpty) last = statusQueue.removeAt(0);
    return last;
  }

  @override
  Future<TelegramBotCredentials> collectCredentials(
    TelegramPairing pairing,
  ) async {
    return const TelegramBotCredentials(
      token: '9001:CHILD-TOKEN',
      botUserId: 9001,
      botUsername: 'pocketclaw_abcd1234_bot',
      ownerUserId: 555,
    );
  }
}

class Fixture {
  Fixture() {
    controller = TelegramOnboardingController(
      client: client,
      configWriter: configWriter,
      reloadCore: () async => reloads++,
      openUrl: (url) async {
        opened.add(url);
        return true;
      },
    );
  }

  final client = StubClient();
  final opened = <String>[];
  TelegramBotCredentials? savedCredentials;
  int reloads = 0;
  late final TelegramOnboardingController controller;

  late final TelegramConfigWriter configWriter = TelegramConfigWriter(
    writeCredentials: (credentials) async {
      savedCredentials = credentials;
      return true;
    },
  );

  Widget widget() => MaterialApp(
    home: TelegramOnboardingPage(
      controller: controller,
      configWriter: configWriter,
    ),
  );
}

Future<void> settle(WidgetTester tester) async {
  for (var i = 0; i < 20; i++) {
    await tester.pump(const Duration(milliseconds: 25));
  }
}

/// Drives the real Android lifecycle sequence. Flutter's binding rejects
/// shortcuts such as paused -> resumed, which is also what a device never
/// does: it always passes through hidden and inactive.
Future<void> background(WidgetTester tester) async {
  for (final state in const [
    AppLifecycleState.inactive,
    AppLifecycleState.hidden,
    AppLifecycleState.paused,
  ]) {
    tester.binding.handleAppLifecycleStateChanged(state);
  }
  await tester.pump();
}

Future<void> foreground(WidgetTester tester) async {
  for (final state in const [
    AppLifecycleState.hidden,
    AppLifecycleState.inactive,
    AppLifecycleState.resumed,
  ]) {
    tester.binding.handleAppLifecycleStateChanged(state);
  }
  await tester.pump();
}

void main() {
  testWidgets('opens on the intro with Connect and a manual fallback', (
    tester,
  ) async {
    final f = Fixture();
    await tester.pumpWidget(f.widget());
    await tester.pump();

    expect(find.text('Connect PocketClaw to Telegram'), findsOneWidget);
    expect(find.text('Connect Telegram'), findsOneWidget);
    expect(find.text('Set up manually'), findsOneWidget);
    // The default path must not put a token field in front of a normal user.
    expect(find.text('Bot token'), findsNothing);
    f.controller.dispose();
  });

  testWidgets('shows Open Telegram and a QR after starting', (tester) async {
    final f = Fixture();
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Connect Telegram'));
    await tester.pump();
    await tester.pump();

    expect(find.text('Open Telegram'), findsOneWidget);
    expect(find.byType(TelegramPairingQr), findsOneWidget);
    expect(find.text('Waiting for Telegram…'), findsOneWidget);
    expect(find.text('@pocketclaw_abcd1234_bot'), findsOneWidget);
    f.controller.dispose();
  });

  testWidgets('the QR encodes the deep link and no secret', (tester) async {
    final f = Fixture();
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Connect Telegram'));
    await tester.pump();
    await tester.pump();

    final qr = tester.widget<TelegramPairingQr>(find.byType(TelegramPairingQr));
    expect(qr.payload, startsWith('https://t.me/newbot/PocketClawSetupBot/'));
    expect(qr.payload, isNot(contains('poll-secret-value')));
    expect(qr.payload, isNot(contains('CHILD-TOKEN')));
    f.controller.dispose();
  });

  testWidgets('tapping Open Telegram launches the deep link', (tester) async {
    final f = Fixture();
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Connect Telegram'));
    await tester.pump();
    await tester.pump();

    await tester.tap(find.text('Open Telegram'));
    await tester.pump();

    expect(f.opened, hasLength(1));
    expect(f.opened.single, contains('t.me/newbot/PocketClawSetupBot/'));
    f.controller.dispose();
  });

  testWidgets('reaches Connected and offers Open Chat', (tester) async {
    final f = Fixture();
    f.client.statusQueue.add(
      const TelegramPairingStatus(
        state: PairingState.ready,
        botUsername: 'pocketclaw_abcd1234_bot',
      ),
    );
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Connect Telegram'));
    await settle(tester);

    expect(find.text('Connected'), findsOneWidget);
    expect(find.text('Open Chat'), findsOneWidget);
    expect(find.text('Done'), findsOneWidget);
    expect(find.text('@pocketclaw_abcd1234_bot'), findsOneWidget);
    // The token configured Core; it is never shown.
    expect(find.textContaining('CHILD-TOKEN'), findsNothing);
    expect(f.savedCredentials?.token, '9001:CHILD-TOKEN');
    expect(f.reloads, 1);
    f.controller.dispose();
  });

  testWidgets('Open Chat opens the new bot conversation', (tester) async {
    final f = Fixture();
    f.client.statusQueue.add(
      const TelegramPairingStatus(
        state: PairingState.ready,
        botUsername: 'pocketclaw_abcd1234_bot',
      ),
    );
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Connect Telegram'));
    await settle(tester);

    await tester.tap(find.text('Open Chat'));
    await tester.pump();

    expect(f.opened.last, 'https://t.me/pocketclaw_abcd1234_bot');
    f.controller.dispose();
  });

  testWidgets('shows the bot-created step while configuring', (tester) async {
    final f = Fixture();
    f.client.statusQueue.add(
      const TelegramPairingStatus(
        state: PairingState.created,
        botUsername: 'pocketclaw_abcd1234_bot',
      ),
    );
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Connect Telegram'));
    await settle(tester);

    expect(find.text('Bot created'), findsOneWidget);
    f.controller.dispose();
  });

  testWidgets('an expired pairing offers a retry', (tester) async {
    final f = Fixture();
    f.client.statusQueue.add(
      const TelegramPairingStatus(state: PairingState.expired),
    );
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Connect Telegram'));
    await settle(tester);

    expect(find.text('Setup link expired'), findsOneWidget);
    expect(find.widgetWithText(FilledButton, 'Try again'), findsOneWidget);
    f.controller.dispose();
  });

  testWidgets('a network failure explains itself and offers both ways out', (
    tester,
  ) async {
    final f = Fixture();
    f.client.createError = const TelegramOnboardingException(
      TelegramOnboardingErrorKind.network,
    );
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Connect Telegram'));
    await settle(tester);

    expect(
      find.textContaining('could not reach the setup service'),
      findsOneWidget,
    );
    expect(find.widgetWithText(FilledButton, 'Try again'), findsOneWidget);
    expect(find.text('Set up manually'), findsOneWidget);
    f.controller.dispose();
  });

  testWidgets('manual setup writes the same Telegram configuration', (
    tester,
  ) async {
    final f = Fixture();
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Set up manually'));
    await tester.pumpAndSettle();

    expect(find.text('Manual Telegram setup'), findsOneWidget);
    await tester.enterText(
      find.widgetWithText(TextField, 'Bot token'),
      '1234567890:AAEXAMPLE-not-a-real-bot-token-for-tests',
    );
    await tester.enterText(
      find.widgetWithText(TextField, 'Owner Telegram numeric user ID'),
      '777',
    );
    await tester.tap(find.text('Save and connect'));
    await tester.pumpAndSettle();

    expect(
      f.savedCredentials?.token,
      '1234567890:AAEXAMPLE-not-a-real-bot-token-for-tests',
    );
    expect(f.savedCredentials?.ownerUserId, 777);
    expect(f.savedCredentials?.botUsername, 'manual');
    f.controller.dispose();
  });

  testWidgets('manual setup rejects a malformed token', (tester) async {
    final f = Fixture();
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Set up manually'));
    await tester.pumpAndSettle();

    await tester.enterText(
      find.widgetWithText(TextField, 'Bot token'),
      'not-a-token',
    );
    await tester.tap(find.text('Save and connect'));
    await tester.pumpAndSettle();

    expect(
      find.textContaining('does not look like a bot token'),
      findsOneWidget,
    );
    expect(
      f.savedCredentials,
      isNull,
      reason: 'nothing should have been written',
    );
    f.controller.dispose();
  });

  testWidgets('manual setup requires a token', (tester) async {
    final f = Fixture();
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Set up manually'));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Save and connect'));
    await tester.pumpAndSettle();

    expect(find.textContaining('Enter the bot token'), findsOneWidget);
    expect(f.savedCredentials, isNull);
    f.controller.dispose();
  });

  testWidgets('manual setup requires a numeric owner before saving', (
    tester,
  ) async {
    final f = Fixture();
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Set up manually'));
    await tester.pumpAndSettle();

    await tester.enterText(
      find.widgetWithText(TextField, 'Bot token'),
      '1234567890:AAEXAMPLE-not-a-real-bot-token-for-tests',
    );
    await tester.tap(find.text('Save and connect'));
    await tester.pumpAndSettle();

    expect(find.textContaining('numeric Telegram user ID'), findsOneWidget);
    expect(f.savedCredentials, isNull);
    f.controller.dispose();
  });

  testWidgets('the manual token field is obscured', (tester) async {
    final f = Fixture();
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Set up manually'));
    await tester.pumpAndSettle();

    final field = tester.widget<TextField>(
      find.widgetWithText(TextField, 'Bot token'),
    );
    expect(field.obscureText, isTrue);
    f.controller.dispose();
  });

  testWidgets('backgrounding the app does not lose the pairing', (
    tester,
  ) async {
    final f = Fixture();
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Connect Telegram'));
    await tester.pump();
    await tester.pump();

    final pairingId = f.controller.pairing!.pairingId;

    await background(tester);
    expect(f.controller.isPolling, isFalse);
    expect(f.controller.pairing!.pairingId, pairingId);
    expect(find.text('Open Telegram'), findsOneWidget);

    await foreground(tester);
    expect(f.controller.isPolling, isTrue);
    expect(f.controller.pairing!.pairingId, pairingId);
    f.controller.dispose();
  });

  testWidgets('returning from Telegram completes the pairing', (tester) async {
    final f = Fixture();
    await tester.pumpWidget(f.widget());
    await tester.tap(find.text('Connect Telegram'));
    await tester.pump();
    await tester.pump();

    await background(tester);

    // While the user was in Telegram, the bot got created.
    f.client.statusQueue.add(
      const TelegramPairingStatus(
        state: PairingState.ready,
        botUsername: 'pocketclaw_abcd1234_bot',
      ),
    );
    await foreground(tester);
    await settle(tester);

    expect(find.text('Connected'), findsOneWidget);
    f.controller.dispose();
  });
}
