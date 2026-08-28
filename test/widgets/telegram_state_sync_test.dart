import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_strings.dart';
import 'package:pocketclaw/src/ui/telegram_settings_card.dart';

void main() {
  group('native Telegram Settings shortcut', () {
    testWidgets('is neutral and never duplicates connection state', (
      tester,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(body: TelegramSettingsCard(onManage: (_) async {})),
        ),
      );

      expect(find.text(TelegramOnboardingStrings.title), findsOneWidget);
      expect(find.text('Manage Telegram connection'), findsOneWidget);
      expect(find.textContaining('Connected'), findsNothing);
      expect(find.text(TelegramOnboardingStrings.introHeadline), findsNothing);
    });

    testWidgets('tap invokes only canonical-console navigation', (
      tester,
    ) async {
      var consoleNavigations = 0;
      var pairingLaunches = 0;
      String? openedPath;

      Future<void> openCanonicalConsole(String path) async {
        consoleNavigations++;
        openedPath = path;
      }

      // No pairing callback exists in TelegramSettingsCard's API. The only
      // injected action is navigation to the authoritative Core console.
      void startPairing() => pairingLaunches++;
      expect(startPairing, isNotNull);

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: TelegramSettingsCard(onManage: openCanonicalConsole),
          ),
        ),
      );
      await tester.tap(find.byKey(const Key('telegram-settings-card')));
      await tester.pumpAndSettle();

      expect(consoleNavigations, 1);
      expect(openedPath, canonicalTelegramConsolePath);
      expect(pairingLaunches, 0);
    });
  });
}
