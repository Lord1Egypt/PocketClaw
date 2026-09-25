import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/config_page.dart';
import 'package:pocketclaw/src/ui/dashboard_page.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  late ServiceManager service;

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    service = ServiceManager();
    await service.setPublicModeConfig(false);
  });

  tearDown(() async {
    await service.setPublicModeConfig(false);
  });

  Future<void> pumpConfig(WidgetTester tester) async {
    await tester.pumpWidget(
      ChangeNotifierProvider<ServiceManager>.value(
        value: service,
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(body: ConfigPage()),
        ),
      ),
    );
    await tester.pump();
  }

  testWidgets(
    'public mode shows a connectable URL and refreshes on IP change',
    (tester) async {
      await service.setPublicModeConfig(true);
      service.setLanAddressForTest('192.168.1.37');
      await pumpConfig(tester);

      expect(find.text('http://192.168.1.37:18800'), findsOneWidget);
      expect(find.text('0.0.0.0'), findsNothing);

      service.setLanAddressForTest('10.0.0.24');
      await tester.pump();

      expect(find.text('http://10.0.0.24:18800'), findsOneWidget);
      expect(find.text('http://192.168.1.37:18800'), findsNothing);

      await service.setPublicModeConfig(false);
    },
  );

  testWidgets('public mode reports no LAN address instead of inventing one', (
    tester,
  ) async {
    await service.setPublicModeConfig(true);
    service.setLanAddressForTest(null);
    await pumpConfig(tester);

    expect(find.text('No LAN address available'), findsOneWidget);
    expect(find.text('0.0.0.0'), findsNothing);

    await service.setPublicModeConfig(false);
  });

  testWidgets('public mode toggle disables input and shows apply progress', (
    tester,
  ) async {
    var toggleCount = 0;
    final focusNode = FocusNode();
    addTearDown(focusNode.dispose);

    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: PublicModeToggle(
            focusNode: focusNode,
            isPublicMode: false,
            isApplying: true,
            onToggle: () => toggleCount++,
            onArrowDown: () {},
            onArrowUp: () {},
          ),
        ),
      ),
    );

    expect(find.text('Applying network mode...'), findsOneWidget);
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    await tester.tap(find.byType(PublicModeToggle));
    expect(toggleCount, 0);
  });

  testWidgets('LAN IP change refreshes the Dashboard QR payload', (
    tester,
  ) async {
    await service.setPublicModeConfig(true);
    service.setLanAddressForTest('192.168.1.37');
    await tester.pumpWidget(
      ChangeNotifierProvider<ServiceManager>.value(
        value: service,
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: DashboardPage(),
        ),
      ),
    );
    await tester.pump();

    expect(
      tester
          .widget<DashboardAccessQrCode>(find.byType(DashboardAccessQrCode))
          .data,
      'http://192.168.1.37:18800',
    );

    service.setLanAddressForTest('10.0.0.24');
    await tester.pump();
    expect(
      tester
          .widget<DashboardAccessQrCode>(find.byType(DashboardAccessQrCode))
          .data,
      'http://10.0.0.24:18800',
    );

    await service.setPublicModeConfig(false);
    await tester.pump();
  });
}
