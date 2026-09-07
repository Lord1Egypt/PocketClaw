import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/dashboard_page.dart';
import 'package:pocketclaw/src/ui/status_sections.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// The endpoint, the QR code and the steps for using them are one access
/// section.
///
/// The steps used to render at the very bottom of the page, after the Status
/// metrics, where they read as a stray footnote rather than as instructions for
/// the QR they describe. These tests pin them back beside the QR.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  late ServiceManager service;

  setUp(() async {
    SharedPreferences.setMockInitialValues({});
    service = ServiceManager();
    await service.updateConfig('127.0.0.1', 18800, publicMode: false);
  });

  tearDown(() async {
    await service.updateConfig('127.0.0.1', 18800, publicMode: false);
  });

  Future<void> pumpDashboard(
    WidgetTester tester, {
    Size size = const Size(400, 1400),
    Locale locale = const Locale('en'),
  }) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(
      ChangeNotifierProvider<ServiceManager>.value(
        value: service,
        child: MaterialApp(
          locale: locale,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const DashboardPage(),
        ),
      ),
    );
    await tester.pump();
  }

  /// The instructions, whichever variant Public Mode selects.
  Finder hintFinder(WidgetTester tester) {
    final l10n = AppLocalizations.of(
      tester.element(find.byType(DashboardPage)),
    )!;
    return find.text(
      service.publicMode ? l10n.publicModeHint : l10n.localModeHint,
    );
  }

  testWidgets('the QR instructions render exactly once', (tester) async {
    await pumpDashboard(tester);
    expect(hintFinder(tester), findsOneWidget);
  });

  testWidgets('the instructions sit inside the same card as the QR', (
    tester,
  ) async {
    await pumpDashboard(tester);

    final hint = hintFinder(tester);
    expect(hint, findsOneWidget);
    expect(find.byType(QrImageView), findsOneWidget);

    // The QR and the instructions share an ancestor that the Status metrics
    // are not part of — that shared ancestor is the access card.
    final sharedAncestors = find
        .ancestor(of: hint, matching: find.byType(Container))
        .evaluate()
        .toSet()
        .intersection(
          find
              .ancestor(of: find.byType(QrImageView), matching: find.byType(Container))
              .evaluate()
              .toSet(),
        );
    expect(
      sharedAncestors,
      isNotEmpty,
      reason: 'the instructions are not inside the QR access card',
    );
  });

  testWidgets('the instructions are not a footer below the Status metrics', (
    tester,
  ) async {
    await pumpDashboard(tester);

    final hintY = tester.getTopLeft(hintFinder(tester)).dy;
    final qrY = tester.getTopLeft(find.byType(QrImageView)).dy;
    final statusY = tester.getTopLeft(find.byType(StatusSections)).dy;

    expect(
      hintY,
      greaterThan(qrY),
      reason: 'the instructions should follow the QR they describe',
    );
    expect(
      hintY,
      lessThan(statusY),
      reason: 'the instructions must not render after the Status metrics',
    );
  });

  testWidgets('Public Mode selects its own instructions, still just once', (
    tester,
  ) async {
    await service.updateConfig('0.0.0.0', 18800, publicMode: true);
    service.setLanAddressForTest('192.168.1.37');
    await pumpDashboard(tester);

    final l10n = AppLocalizations.of(
      tester.element(find.byType(DashboardPage)),
    )!;
    expect(find.text(l10n.publicModeHint), findsOneWidget);
    expect(find.text(l10n.localModeHint), findsNothing);

    // Public Mode starts the LAN-address poll; stop it inside the test body,
    // as the other public-mode tests do, or the framework fails on a pending
    // timer before tearDown runs.
    await service.updateConfig('127.0.0.1', 18800, publicMode: false);
  });

  testWidgets('the Status sections still render below the access card', (
    tester,
  ) async {
    await pumpDashboard(tester);

    expect(find.byType(StatusSections), findsOneWidget);
    final l10n = AppLocalizations.of(
      tester.element(find.byType(DashboardPage)),
    )!;
    // Gateway is stopped in this harness, so System renders and the detailed
    // sections report themselves unavailable rather than showing zeroes.
    expect(find.text(l10n.statusSectionSystem), findsOneWidget);
  });

  testWidgets('the access section holds together in Arabic RTL', (
    tester,
  ) async {
    await pumpDashboard(tester, locale: const Locale('ar'));

    expect(
      Directionality.of(tester.element(find.byType(DashboardPage))),
      TextDirection.rtl,
    );
    expect(hintFinder(tester), findsOneWidget);

    final hintY = tester.getTopLeft(hintFinder(tester)).dy;
    final statusY = tester.getTopLeft(find.byType(StatusSections)).dy;
    expect(hintY, lessThan(statusY));
  });

  navigationGuards();

  testWidgets('the wide layout keeps the instructions in the card too', (
    tester,
  ) async {
    await pumpDashboard(tester, size: const Size(900, 1400));

    final hintY = tester.getTopLeft(hintFinder(tester)).dy;
    final statusY = tester.getTopLeft(find.byType(StatusSections)).dy;
    expect(hintFinder(tester), findsOneWidget);
    expect(hintY, lessThan(statusY));
  });
}

/// The access-section fix must not have disturbed navigation. PocketClaw has
/// four destinations — Status/Dashboard, Web, Logs, Settings — and Status stays
/// integrated into the first one rather than becoming a destination of its own.
///
/// This reads main.dart rather than pumping the shell: MainShell binds tray and
/// window listeners and a ServiceManager on construction, so standing it up in
/// a widget test would assert more about those than about the navigation. A
/// count of the destination builders is the fact worth pinning, and adding or
/// removing one fails here.
void navigationGuards() {
  test('navigation still declares exactly four destinations', () {
    final source = File('lib/main.dart').readAsStringSync();
    // Calls only: the builder's own declaration is `_buildNavButton({`, so a
    // bare `_buildNavButton(` match would count it as a fifth destination.
    final destinations = RegExp(
      r'_buildNavButton\((?!\{)',
    ).allMatches(source).length;

    expect(
      destinations,
      4,
      reason:
          'main.dart declares $destinations navigation destinations; Status is '
          'meant to stay integrated into the Dashboard destination, not become '
          'a fifth tab',
    );
    // The Dashboard destination is the one Status lives in.
    expect(source.contains('DashboardPage(detailEnabled:'), isTrue);
  });
}
