import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/core/app_theme.dart';
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
    Locale? locale,
  }) async {
    final service = ServiceManager();

    await tester.pumpWidget(
      ChangeNotifierProvider<ServiceManager>.value(
        value: service,
        child: MaterialApp(
          locale: locale,
          theme: ApertureTheme.dark(AppThemeMode.carbon),
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

  Future<void> openAbout(WidgetTester tester) async {
    await tester.tap(find.byIcon(Icons.info_outline));
    await tester.pumpAndSettle();
  }

  AppLocalizations dialogL10n(WidgetTester tester) =>
      AppLocalizations.of(tester.element(find.byType(AlertDialog)))!;

  testWidgets('About is a dialog and Close dismisses it', (
    WidgetTester tester,
  ) async {
    await pumpConfigPage(
      tester,
      aboutInfoLoader: () async =>
          const AboutInfo(appVersion: '1.0.0', coreVersion: 'core-1.0.0'),
    );

    expect(find.byType(AlertDialog), findsNothing);

    await openAbout(tester);

    expect(find.byType(AlertDialog), findsOneWidget);
    expect(
      find.text('PocketClaw is your private AI assistant workspace.'),
      findsOneWidget,
    );

    await tester.tap(find.text('Close'));
    await tester.pumpAndSettle();

    expect(find.byType(AlertDialog), findsNothing);
    expect(find.text('PocketClaw'), findsNothing);
  });

  testWidgets('shows PocketClaw identity and the loader-supplied versions', (
    WidgetTester tester,
  ) async {
    await pumpConfigPage(
      tester,
      aboutInfoLoader: () async =>
          const AboutInfo(appVersion: '1.2.3', coreVersion: 'core-9.8.7'),
    );

    await openAbout(tester);

    expect(find.text('PocketClaw'), findsOneWidget);
    expect(find.text('PocketClaw version'), findsOneWidget);
    expect(find.text('1.2.3'), findsOneWidget);
    expect(find.text('Runtime version'), findsOneWidget);
    expect(find.text('core-9.8.7'), findsOneWidget);
    expect(find.textContaining('PicoClaw'), findsNothing);
    expect(find.text('Sipeed'), findsNothing);
    expect(find.byTooltip('GitHub'), findsNothing);
  });

  // The versions are read at runtime from the platform and from Core. If a
  // release number were ever typed into the dialog it would keep rendering
  // after the release moved on, which is worse than showing nothing.
  testWidgets('renders no version the loader did not supply', (
    WidgetTester tester,
  ) async {
    await pumpConfigPage(
      tester,
      aboutInfoLoader: () async => const AboutInfo(
        appVersion: 'app-from-loader',
        coreVersion: 'core-from-loader',
      ),
    );

    await openAbout(tester);

    expect(find.text('app-from-loader'), findsOneWidget);
    expect(find.text('core-from-loader'), findsOneWidget);
    for (final shipped in <String>['0.2.0', '0.3.1', 'v0.2.0', 'v0.3.1']) {
      expect(
        find.textContaining(shipped),
        findsNothing,
        reason: 'About must not carry a hardcoded release number',
      );
    }
  });

  testWidgets('draws the version block with Aperture surfaces and tokens', (
    WidgetTester tester,
  ) async {
    await pumpConfigPage(
      tester,
      aboutInfoLoader: () async =>
          const AboutInfo(appVersion: '1.0.0', coreVersion: 'core-1.0.0'),
    );

    await openAbout(tester);

    final bracket = find.descendant(
      of: find.byType(AlertDialog),
      matching: find.byType(ApertureBracket),
    );
    expect(bracket, findsOneWidget);

    final tokens = tester.element(find.byType(AlertDialog)).aperture;
    expect(tester.widget<ApertureBracket>(bracket).fill, tokens.surface1);
    expect(tester.widget<ApertureBracket>(bracket).borderColor, tokens.border);

    // The dismiss control is the themed filled treatment, not a bare text
    // button carrying the framework default.
    expect(
      find.descendant(
        of: find.byType(AlertDialog),
        matching: find.byType(FilledButton),
      ),
      findsOneWidget,
    );
  });

  testWidgets('does not resize the dialog when the versions arrive', (
    WidgetTester tester,
  ) async {
    final aboutInfoCompleter = Completer<AboutInfo>();

    await pumpConfigPage(
      tester,
      aboutInfoLoader: () => aboutInfoCompleter.future,
      settle: false,
    );

    await tester.tap(find.byIcon(Icons.info_outline));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));

    // Labels are present before the values are, so the block already occupies
    // its final height.
    expect(find.text('PocketClaw version'), findsOneWidget);
    expect(find.text('Runtime version'), findsOneWidget);
    final loadingSize = tester.getSize(find.byType(AlertDialog));

    aboutInfoCompleter.complete(
      const AboutInfo(appVersion: '1.0.0', coreVersion: 'core-1.0.0'),
    );
    await tester.pumpAndSettle();

    final loadedSize = tester.getSize(find.byType(AlertDialog));
    expect(find.text('1.0.0'), findsOneWidget);
    expect(find.text('core-1.0.0'), findsOneWidget);
    expect(
      (loadedSize.height - loadingSize.height).abs(),
      lessThan(4.0),
      reason: 'loading to loaded must not jump the dialog height',
    );
  });

  testWidgets('uses the production about loader when no override is provided', (
    WidgetTester tester,
  ) async {
    await pumpConfigPage(tester);

    await tester.tap(find.byIcon(Icons.info_outline));
    await tester.pump();

    // Nothing was injected, so the values are still pending and the labels
    // stand alone.
    expect(find.text('PocketClaw version'), findsOneWidget);
    expect(find.text('Runtime version'), findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(AlertDialog),
        matching: find.byType(CircularProgressIndicator),
      ),
      findsNWidgets(2),
    );
  });

  testWidgets('shows localized fallback text when a version is unavailable', (
    WidgetTester tester,
  ) async {
    await pumpConfigPage(
      tester,
      aboutInfoLoader: () async =>
          const AboutInfo(appVersion: '', coreVersion: 'unknown'),
    );

    await openAbout(tester);

    expect(
      find.text(dialogL10n(tester).aboutVersionUnavailable),
      findsNWidgets(2),
    );
  });

  // PC-DEF-063. A failed Core probe is an absence (null), which is a different
  // state from the future still running. The dialog must render Unavailable for
  // the completed-failure case and never the loading spinner.
  testWidgets('renders Unavailable when the Core probe returned null', (
    WidgetTester tester,
  ) async {
    await pumpConfigPage(
      tester,
      aboutInfoLoader: () async =>
          const AboutInfo(appVersion: '1.2.3', coreVersion: null),
    );

    await openAbout(tester);

    expect(find.text('1.2.3'), findsOneWidget);
    expect(
      find.text(dialogL10n(tester).aboutVersionUnavailable),
      findsOneWidget,
    );
    expect(
      find.descendant(
        of: find.byType(AlertDialog),
        matching: find.byType(CircularProgressIndicator),
      ),
      findsNothing,
      reason: 'a completed failed probe is not a loading state',
    );
  });

  testWidgets('lays out on a narrow phone without overflowing', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(320, 640));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    await pumpConfigPage(
      tester,
      aboutInfoLoader: () async => const AboutInfo(
        appVersion: '0.0.0+1 (aaaaaaaaaaaa)',
        coreVersion: 'runtime v0.0.0 (aaaaaaaaaaaa)',
      ),
    );

    await openAbout(tester);

    expect(find.byType(AlertDialog), findsOneWidget);
    expect(find.text('PocketClaw version'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('mirrors for Arabic while keeping version values left-to-right', (
    WidgetTester tester,
  ) async {
    await pumpConfigPage(
      tester,
      locale: const Locale('ar'),
      aboutInfoLoader: () async =>
          const AboutInfo(appVersion: '1.2.3', coreVersion: 'core-9.8.7'),
    );

    await openAbout(tester);

    final l10n = dialogL10n(tester);
    expect(
      Directionality.of(tester.element(find.byType(AlertDialog))),
      TextDirection.rtl,
    );
    expect(find.text(l10n.aboutAppVersionLabel), findsOneWidget);
    expect(find.text(l10n.aboutCoreVersionLabel), findsOneWidget);

    // A version compared character by character must not reverse.
    final valueDirection = tester.widget<Directionality>(
      find
          .ancestor(
            of: find.text('core-9.8.7'),
            matching: find.byType(Directionality),
          )
          .first,
    );
    expect(valueDirection.textDirection, TextDirection.ltr);
    expect(tester.takeException(), isNull);
  });
}
