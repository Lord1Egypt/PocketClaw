import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/config_page.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues({});
  });

  Future<void> pumpConfigPage(
    WidgetTester tester, {
    Future<AboutInfo> Function()? aboutInfoLoader,
    bool settle = true,
  }) async {
    final service = ServiceManager();

    await tester.pumpWidget(
      ChangeNotifierProvider<ServiceManager>.value(
        value: service,
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(body: ConfigPage(aboutInfoLoader: aboutInfoLoader)),
        ),
      ),
    );

    if (settle) {
      await tester.pumpAndSettle();
    }
  }

  testWidgets('opens and closes the about dialog from settings', (
    WidgetTester tester,
  ) async {
    await pumpConfigPage(
      tester,
      aboutInfoLoader: () async =>
          const AboutInfo(appVersion: '1.0.0', coreVersion: 'core-1.0.0'),
    );

    expect(find.text('About'), findsOneWidget);

    await tester.tap(find.text('About'));
    await tester.pumpAndSettle();

    expect(find.text('About'), findsWidgets);
    expect(
      find.text('PocketClaw is your private AI assistant workspace.'),
      findsOneWidget,
    );

    await tester.tap(find.text('Close'));
    await tester.pumpAndSettle();

    expect(find.text('PocketClaw'), findsNothing);
  });

  testWidgets('shows PocketClaw identity without upstream branding', (
    WidgetTester tester,
  ) async {
    await pumpConfigPage(
      tester,
      aboutInfoLoader: () async =>
          const AboutInfo(appVersion: '1.2.3', coreVersion: 'core-9.8.7'),
    );

    await tester.tap(find.text('About'));
    await tester.pumpAndSettle();

    expect(find.text('About'), findsWidgets);
    expect(find.text('PocketClaw'), findsOneWidget);
    expect(find.text('PocketClaw version'), findsOneWidget);
    expect(find.text('1.2.3'), findsOneWidget);
    expect(find.text('Runtime version'), findsOneWidget);
    expect(find.text('core-9.8.7'), findsOneWidget);
    expect(find.textContaining('PicoClaw'), findsNothing);
    expect(find.text('Sipeed'), findsNothing);
    expect(find.byTooltip('GitHub'), findsNothing);
  });

  testWidgets('shows loading indicator while about info is still loading', (
    WidgetTester tester,
  ) async {
    final aboutInfoCompleter = Completer<AboutInfo>();

    await pumpConfigPage(
      tester,
      aboutInfoLoader: () => aboutInfoCompleter.future,
      settle: false,
    );

    await tester.tap(find.text('About'));
    await tester.pump();

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(find.text('PocketClaw version'), findsNothing);
    expect(find.text('Runtime version'), findsNothing);

    aboutInfoCompleter.complete(
      const AboutInfo(appVersion: '1.0.0', coreVersion: 'core-1.0.0'),
    );
    await tester.pumpAndSettle();

    expect(find.byType(CircularProgressIndicator), findsNothing);
    expect(find.text('PocketClaw version'), findsOneWidget);
    expect(find.text('1.0.0'), findsOneWidget);
    expect(find.text('Runtime version'), findsOneWidget);
    expect(find.text('core-1.0.0'), findsOneWidget);
  });

  testWidgets('shows localized fallback text when a version is unavailable', (
    WidgetTester tester,
  ) async {
    await pumpConfigPage(
      tester,
      aboutInfoLoader: () async =>
          const AboutInfo(appVersion: '', coreVersion: 'unknown'),
    );

    await tester.tap(find.text('About'));
    await tester.pumpAndSettle();

    final l10n = AppLocalizations.of(tester.element(find.byType(AlertDialog)))!;
    expect(find.text(l10n.aboutVersionUnavailable), findsNWidgets(2));
  });

  testWidgets('uses production about loader when no override is provided', (
    WidgetTester tester,
  ) async {
    await pumpConfigPage(tester);

    await tester.tap(find.text('About'));
    await tester.pump();

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
  });
}
