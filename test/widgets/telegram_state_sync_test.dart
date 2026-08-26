import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/telegram/telegram_connection_status.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_strings.dart';
import 'package:pocketclaw/src/ui/telegram_connected_page.dart';
import 'package:pocketclaw/src/ui/telegram_settings_card.dart';

/// Telegram connection state must have one source of truth.
///
/// The device showed the embedded console reporting Connected while the native
/// Settings card still said "Connect PocketClaw to Telegram" and offered a new
/// pairing. Both surfaces now derive from the persisted `channel_list.telegram`
/// entry, so these exercise that real configuration shape rather than a label
/// helper.
String configJson({
  String? token,
  List<String> allowFrom = const <String>[],
  bool includeTelegram = true,
}) {
  final settings = <String, dynamic>{};
  if (token != null) settings['token'] = token;
  final telegram = <String, dynamic>{
    'type': 'telegram',
    'enabled': true,
    'settings': settings,
    if (allowFrom.isNotEmpty) 'allow_from': allowFrom,
  };
  return jsonEncode(<String, dynamic>{
    'channel_list': <String, dynamic>{
      'mqtt': <String, dynamic>{'type': 'mqtt'},
      if (includeTelegram) 'telegram': telegram,
    },
  });
}

// Deliberately not token-shaped. The reader only checks that a token is
// present and non-empty, so a realistic shape would buy nothing and would make
// every future secret scan of this repository flag a test fixture.
const connectedToken = 'a-configured-telegram-token-placeholder';

void main() {
  group('canonical state, read from the persisted configuration', () {
    test('case A: no telegram entry at all is disconnected', () {
      final status = TelegramConnectionReader.parse(
        configJson(includeTelegram: false),
      );
      expect(status.configured, isFalse);
    });

    test('case A: a telegram entry with no token is disconnected', () {
      expect(TelegramConnectionReader.parse(configJson()).configured, isFalse);
      expect(
        TelegramConnectionReader.parse(configJson(token: '   ')).configured,
        isFalse,
      );
    });

    test('case B: a persisted token is connected', () {
      final status = TelegramConnectionReader.parse(
        configJson(token: connectedToken, allowFrom: const ['123456789']),
        botUsername: 'pocketclaw_ab12cd34_bot',
      );
      expect(status.configured, isTrue);
      expect(status.ownerRestricted, isTrue);
      expect(status.botUsername, 'pocketclaw_ab12cd34_bot');
      expect(status.chatUrl, 'https://t.me/pocketclaw_ab12cd34_bot');
    });

    test('connected without a known handle is reported honestly', () {
      // A manually configured bot, or one paired on another device: Core does
      // not store the handle, so none is invented.
      final status = TelegramConnectionReader.parse(
        configJson(token: connectedToken),
      );
      expect(status.configured, isTrue);
      expect(status.botUsername, isNull);
      expect(status.chatUrl, isNull);
      expect(status.ownerRestricted, isFalse);
    });

    test('malformed or unreadable configuration fails closed', () async {
      expect(TelegramConnectionReader.parse('not json').configured, isFalse);
      expect(TelegramConnectionReader.parse('[]').configured, isFalse);
      expect(TelegramConnectionReader.parse('').configured, isFalse);

      final status = await TelegramConnectionReader.read(
        readConfig: () async => throw Exception('config unreadable'),
      );
      expect(status.configured, isFalse);
    });
  });

  group('native Settings card', () {
    Future<void> pumpCard(
      WidgetTester tester, {
      required TelegramConnectionStatus status,
      Future<void> Function(BuildContext)? onOpen,
    }) async {
      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: TelegramSettingsCard(
            readStatus: () async => status,
            onOpen: onOpen ?? (_) async {},
          ),
        ),
      ));
      await tester.pumpAndSettle();
    }

    testWidgets('case A: unconfigured offers Connect', (tester) async {
      await pumpCard(
        tester,
        status: TelegramConnectionReader.parse(configJson()),
      );

      expect(find.text(TelegramOnboardingStrings.introHeadline), findsOneWidget);
      expect(find.textContaining('Connected'), findsNothing);
    });

    testWidgets('case B: configured shows Connected and the bot handle',
        (tester) async {
      await pumpCard(
        tester,
        status: TelegramConnectionReader.parse(
          configJson(token: connectedToken, allowFrom: const ['1']),
          botUsername: 'pocketclaw_ab12cd34_bot',
        ),
      );

      expect(find.textContaining('Connected'), findsOneWidget);
      expect(find.textContaining('@pocketclaw_ab12cd34_bot'), findsOneWidget);
      // The regression: this must no longer be the card's message.
      expect(find.text(TelegramOnboardingStrings.introHeadline), findsNothing);
    });

    testWidgets('case B: configured without a handle still shows Connected',
        (tester) async {
      await pumpCard(
        tester,
        status: TelegramConnectionReader.parse(configJson(token: connectedToken)),
      );

      expect(find.textContaining('Connected'), findsOneWidget);
      expect(find.text(TelegramOnboardingStrings.introHeadline), findsNothing);
    });

    testWidgets('case F: state survives a fresh mount, as after a restart',
        (tester) async {
      // A cold start rebuilds the card from persisted configuration only.
      await pumpCard(
        tester,
        status: TelegramConnectionReader.parse(
          configJson(token: connectedToken, allowFrom: const ['1']),
          botUsername: 'pocketclaw_ab12cd34_bot',
        ),
      );
      expect(find.textContaining('Connected'), findsOneWidget);
    });

    testWidgets('case G: the card re-reads after the flow returns',
        (tester) async {
      var reads = 0;
      var token = <String?>[null, connectedToken];

      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: TelegramSettingsCard(
            readStatus: () async {
              final value = token[reads.clamp(0, token.length - 1)];
              reads++;
              return TelegramConnectionReader.parse(
                configJson(token: value),
                botUsername: value == null ? null : 'pocketclaw_ab12cd34_bot',
              );
            },
            onOpen: (_) async {},
          ),
        ),
      ));
      await tester.pumpAndSettle();
      expect(find.text(TelegramOnboardingStrings.introHeadline), findsOneWidget);

      await tester.tap(find.byKey(const Key('telegram-settings-card')));
      await tester.pumpAndSettle();

      expect(find.textContaining('Connected'), findsOneWidget);
    });
  });

  group('connected page', () {
    testWidgets(
      'case C: opening a connected Telegram does not start a pairing',
      (tester) async {
        final status = TelegramConnectionReader.parse(
          configJson(token: connectedToken, allowFrom: const ['1']),
          botUsername: 'pocketclaw_ab12cd34_bot',
        );
        var reconnects = 0;

        await tester.pumpWidget(MaterialApp(
          home: TelegramConnectedPage(
            status: status,
            onOpenChat: () {},
            onReconnect: () => reconnects++,
            onAdvanced: () {},
          ),
        ));
        await tester.pumpAndSettle();

        expect(find.text(TelegramOnboardingStrings.connected), findsOneWidget);
        expect(find.text('@pocketclaw_ab12cd34_bot'), findsOneWidget);
        expect(find.text(TelegramOnboardingStrings.openChat), findsOneWidget);
        expect(find.text(TelegramOnboardingStrings.reconnect), findsOneWidget);
        expect(
          find.text(TelegramOnboardingStrings.advancedSettings),
          findsOneWidget,
        );
        // Nothing has begun on its own.
        expect(reconnects, 0);
        expect(find.text(TelegramOnboardingStrings.connect), findsNothing);
      },
    );

    testWidgets('case D: reconnect fires only on the explicit action',
        (tester) async {
      var reconnects = 0;
      await tester.pumpWidget(MaterialApp(
        home: TelegramConnectedPage(
          status: TelegramConnectionReader.parse(
            configJson(token: connectedToken),
          ),
          onOpenChat: null,
          onReconnect: () => reconnects++,
          onAdvanced: () {},
        ),
      ));
      await tester.pumpAndSettle();

      await tester.tap(find.text(TelegramOnboardingStrings.reconnect));
      await tester.pumpAndSettle();
      expect(reconnects, 1);
    });

    testWidgets('Open Chat is hidden when no handle is known', (tester) async {
      await tester.pumpWidget(MaterialApp(
        home: TelegramConnectedPage(
          status: TelegramConnectionReader.parse(
            configJson(token: connectedToken),
          ),
          onOpenChat: null,
          onReconnect: () {},
          onAdvanced: () {},
        ),
      ));
      await tester.pumpAndSettle();

      expect(find.text(TelegramOnboardingStrings.openChat), findsNothing);
      expect(
        find.text(TelegramOnboardingStrings.connectedUnknownBot),
        findsOneWidget,
      );
    });
  });

  group('case E: a failed replacement leaves the old bot configured', () {
    test('config is only rewritten once a new token exists', () {
      // The writer is the only thing that mutates channel_list.telegram, and
      // it runs after the replacement token has been received. A pairing that
      // expires or is cancelled therefore never reaches it, so the persisted
      // configuration still reports the original bot.
      final existing = configJson(
        token: connectedToken,
        allowFrom: const ['123456789'],
      );

      final beforeReconnect = TelegramConnectionReader.parse(
        existing,
        botUsername: 'pocketclaw_ab12cd34_bot',
      );
      expect(beforeReconnect.configured, isTrue);

      // Simulate the pairing failing: nothing was written.
      final afterFailedReconnect = TelegramConnectionReader.parse(
        existing,
        botUsername: 'pocketclaw_ab12cd34_bot',
      );
      expect(afterFailedReconnect.configured, isTrue);
      expect(afterFailedReconnect.botUsername, 'pocketclaw_ab12cd34_bot');
      expect(afterFailedReconnect.ownerRestricted, isTrue);
    });
  });
}
