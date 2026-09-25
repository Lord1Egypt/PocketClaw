import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/webview/webview_nav_bar.dart';

/// The floating back/forward/reload pill over the Dashboard WebView.
///
/// It used to start 12 px below the top-right corner, on top of the
/// Dashboard's own header: in Arabic that is the sidebar menu button, so a tap
/// meant for the menu went Back instead and the sidebar could not be opened.
void main() {
  Future<Rect> pumpPill(WidgetTester tester, Locale locale) async {
    tester.view.physicalSize = const Size(1080, 2340);
    tester.view.devicePixelRatio = 2.625;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(
      MaterialApp(
        locale: locale,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const Scaffold(
          body: Stack(children: [Positioned.fill(child: DraggableWebNavBar())]),
        ),
      ),
    );
    await tester.pumpAndSettle();
    return tester.getRect(find.byType(Material).last);
  }

  for (final locale in const [Locale('en'), Locale('ar')]) {
    testWidgets(
      'the pill starts clear of the header (${locale.languageCode})',
      (tester) async {
        final pill = await pumpPill(tester, locale);
        final screen = tester.view.physicalSize / tester.view.devicePixelRatio;
        // The Dashboard header and its page-title row fill the top ~110 dp.
        expect(pill.top, greaterThan(110));
        expect(pill.center.dy, closeTo(screen.height / 2, 60));
      },
    );
  }
}
