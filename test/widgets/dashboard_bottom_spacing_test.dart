import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/dashboard_page.dart';
import 'package:pocketclaw/src/ui/status_sections.dart';
import 'package:pocketclaw/src/ui/widgets/adaptive_action_bar.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// The gap between the last card and the navigation bar.
///
/// The page used to end with a 100px spacer reserving room for a bottom bar
/// that never overlaps it: AdaptiveActionBar lays the shell out as
/// SafeArea(Column[Expanded(content), bar]), so the bar is a sibling below the
/// scroll view. The spacer was invisible only while the access hint still
/// rendered inside it; once that moved into the access card it became a dead
/// band under the Resources card.
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

  /// Pumps the real shell: the page inside the navigation bar that sits below
  /// it, so distances are measured against the widget that actually bounds the
  /// content.
  Future<void> pumpShell(
    WidgetTester tester, {
    Size size = const Size(400, 800),
    Locale locale = const Locale('en'),
    double bottomInset = 0,
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
          home: MediaQuery(
            data: MediaQueryData(
              size: size,
              padding: EdgeInsets.only(bottom: bottomInset),
              viewPadding: EdgeInsets.only(bottom: bottomInset),
            ),
            child: AdaptiveActionBar(
              content: const DashboardPage(),
              actions: const [
                Icon(Icons.dashboard_outlined),
                Icon(Icons.language_outlined),
                Icon(Icons.article_outlined),
                Icon(Icons.settings_outlined),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pump();
  }

  /// Scrolls the page to its end so the last card and the trailing gap are
  /// both laid out.
  ///
  /// The position is driven to maxScrollExtent rather than dragged: the sliver
  /// builds its children lazily, so the last card does not exist until the
  /// viewport reaches it, and a fling would leave the result dependent on
  /// bouncing physics settling.
  Future<void> scrollToBottom(WidgetTester tester) async {
    final position = tester
        .state<ScrollableState>(find.byType(Scrollable).first)
        .position;
    position.jumpTo(position.maxScrollExtent);
    await tester.pumpAndSettle();
  }

  /// Distance from the bottom of the last Status card to the bottom edge of
  /// the scrolling viewport.
  ///
  /// Measured against the viewport rather than the navigation bar's icons: the
  /// bar has its own internal padding above them, which would mask a page whose
  /// content runs right to the edge.
  double trailingGap(WidgetTester tester) {
    final contentBottom = tester.getRect(find.byType(StatusSections)).bottom;
    final viewportBottom = tester.getRect(find.byType(CustomScrollView)).bottom;
    return viewportBottom - contentBottom;
  }

  /// Top edge of the navigation bar, taken from the first action icon's row.
  double navBarTop(WidgetTester tester) =>
      tester.getRect(find.byIcon(Icons.dashboard_outlined)).top;

  testWidgets('no obsolete 100px footer remains after the Status sections', (
    tester,
  ) async {
    await pumpShell(tester);
    await scrollToBottom(tester);

    final gap = trailingGap(tester);
    // The dead band was 100px of spacer on top of the scroll view's own 16px
    // bottom padding. Anything approaching that is the footer coming back.
    expect(
      gap,
      lessThan(64),
      reason: 'a large empty band remains under the last card: ${gap}px',
    );
  });

  testWidgets('a small intentional bottom gap survives', (tester) async {
    await pumpShell(tester);
    await scrollToBottom(tester);

    final gap = trailingGap(tester);
    // A real gap inside the page, not merely the navigation bar's own padding.
    expect(
      gap,
      greaterThanOrEqualTo(16),
      reason: 'the last card runs to the very bottom of the page: ${gap}px',
    );
  });

  testWidgets('the last Resources card scrolls fully above the nav bar', (
    tester,
  ) async {
    await pumpShell(tester);
    await scrollToBottom(tester);

    final l10n = AppLocalizations.of(
      tester.element(find.byType(DashboardPage)),
    )!;
    // Gateway is stopped in this harness, so System is the last rendered
    // section; either way the final card must clear the bar.
    final lastSectionLabel = find.text(l10n.statusSectionSystem);
    expect(lastSectionLabel, findsOneWidget);

    expect(
      tester.getRect(find.byType(StatusSections)).bottom,
      lessThanOrEqualTo(navBarTop(tester)),
      reason: 'the final card is hidden behind the navigation bar',
    );
  });

  testWidgets('the safe-area bottom inset is not double-counted', (
    tester,
  ) async {
    await pumpShell(tester);
    await scrollToBottom(tester);
    final gapWithoutInset = trailingGap(tester);

    await pumpShell(tester, bottomInset: 48);
    await scrollToBottom(tester);
    final gapWithInset = trailingGap(tester);

    // AdaptiveActionBar's SafeArea consumes the inset once and removes it from
    // the MediaQuery it exposes, so the page adds none of its own. The gap
    // between the card and the bar is therefore unchanged by a system inset.
    expect(
      gapWithInset,
      closeTo(gapWithoutInset, 1.0),
      reason:
          'the bottom inset changed the in-page gap from $gapWithoutInset to '
          '$gapWithInset, which means it is being applied more than once',
    );
  });

  testWidgets('the narrow Arabic layout keeps the same bottom behaviour', (
    tester,
  ) async {
    await pumpShell(
      tester,
      size: const Size(320, 640),
      locale: const Locale('ar'),
    );
    await scrollToBottom(tester);

    expect(
      Directionality.of(tester.element(find.byType(DashboardPage))),
      TextDirection.rtl,
    );
    expect(tester.takeException(), isNull);

    final gap = trailingGap(tester);
    expect(gap, greaterThanOrEqualTo(16));
    expect(gap, lessThan(64));
    expect(
      tester.getRect(find.byType(StatusSections)).bottom,
      lessThanOrEqualTo(navBarTop(tester)),
    );
  });
}
