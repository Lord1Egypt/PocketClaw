import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/config_page.dart';
import 'package:pocketclaw/src/whats_new/whats_new_seen_store.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// A narrow but entirely ordinary phone. The unseen NEW badge widened the
/// Settings header past this, and the title — not the badge — paid for it.
const Size narrowPhone = Size(360, 800);

class _FakeSeenStore implements WhatsNewSeenStore {
  _FakeSeenStore(this.version);

  String? version;

  @override
  Future<String?> readLastSeenVersion() async => version;

  @override
  Future<void> writeLastSeenVersion(String value) async => version = value;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() => SharedPreferences.setMockInitialValues({}));

  Future<AppLocalizations> pumpSettings(
    WidgetTester tester, {
    required bool unseen,
    Locale locale = const Locale('en'),
  }) async {
    tester.view.physicalSize = narrowPhone;
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(
      ChangeNotifierProvider<ServiceManager>.value(
        value: ServiceManager(),
        child: MaterialApp(
          locale: locale,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: ConfigPage(
              whatsNewSeenStore: _FakeSeenStore(unseen ? null : '0.2.0'),
              whatsNewVersionLoader: () async => '0.2.0',
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    return AppLocalizations.of(tester.element(find.byType(ConfigPage)))!;
  }

  /// The title is truncated exactly when the box it was given is narrower than
  /// the text it wants to paint. Comparing the two catches an ellipsis without
  /// depending on where the renderer chose to cut the word.
  void expectTitleFullyVisible(WidgetTester tester, String title) {
    final paragraph = tester.renderObject<RenderParagraph>(find.text(title));
    final needed = paragraph.getMaxIntrinsicWidth(double.infinity);
    expect(
      paragraph.size.width,
      greaterThanOrEqualTo(needed - 0.5),
      reason:
          'the "$title" header was given ${paragraph.size.width}px but needs '
          '${needed}px, so it renders truncated',
    );
    expect(
      paragraph.size.height,
      lessThan(paragraph.getMaxIntrinsicHeight(double.infinity) * 2),
      reason: 'the title should not have to wrap to fit',
    );
  }

  void expectHeaderComplete(WidgetTester tester, AppLocalizations l10n) {
    expect(find.text(l10n.whatsNewTitle), findsOneWidget);
    expect(find.text(l10n.about), findsOneWidget);
    expectTitleFullyVisible(tester, l10n.settings);
    expect(
      tester.takeException(),
      isNull,
      reason: 'the header must not overflow at $narrowPhone',
    );
  }

  testWidgets('renders the whole header on a narrow phone with NEW showing', (
    tester,
  ) async {
    final l10n = await pumpSettings(tester, unseen: true);

    expect(find.text(l10n.whatsNewBadge), findsOneWidget);
    expectHeaderComplete(tester, l10n);
  });

  testWidgets('renders the whole header on a narrow phone once NEW is seen', (
    tester,
  ) async {
    final l10n = await pumpSettings(tester, unseen: false);

    expect(find.text(l10n.whatsNewBadge), findsNothing);
    expectHeaderComplete(tester, l10n);
  });

  testWidgets('renders the whole Arabic header with NEW showing', (
    tester,
  ) async {
    final l10n = await pumpSettings(
      tester,
      unseen: true,
      locale: const Locale('ar'),
    );

    expect(
      Directionality.of(tester.element(find.byType(ConfigPage))),
      TextDirection.rtl,
    );
    expect(find.text(l10n.whatsNewBadge), findsOneWidget);
    expectHeaderComplete(tester, l10n);
  });

  testWidgets('renders the whole Arabic header once NEW is seen', (
    tester,
  ) async {
    final l10n = await pumpSettings(
      tester,
      unseen: false,
      locale: const Locale('ar'),
    );

    expect(find.text(l10n.whatsNewBadge), findsNothing);
    expectHeaderComplete(tester, l10n);
  });
}
