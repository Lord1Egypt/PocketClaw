import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/whats_new_page.dart';
import 'package:pocketclaw/src/whats_new/whats_new_release.dart';

/// Words this release must never advertise. Some name work that was removed,
/// some name work that was abandoned, and some are internal build detail no
/// user should read in release notes.
const List<String> forbiddenSubstrings = <String>[
  'WhatsApp',
  'BlueStacks',
  'Auto-Start',
  'Auto Start',
  'versionCode',
];

void main() {
  Future<AppLocalizations> pumpWhatsNew(
    WidgetTester tester, {
    Locale locale = const Locale('en'),
  }) async {
    // The notes are a lazily built list; a phone-sized viewport would leave
    // the later sections unbuilt and silently unasserted.
    tester.view.physicalSize = const Size(1000, 2400);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(
      MaterialApp(
        locale: locale,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: const WhatsNewPage(),
      ),
    );
    await tester.pumpAndSettle();
    return AppLocalizations.of(tester.element(find.byType(WhatsNewPage)))!;
  }

  testWidgets('renders the 0.2.0 release header and every section', (
    tester,
  ) async {
    final l10n = await pumpWhatsNew(tester);

    expect(find.text('PocketClaw 0.2.0'), findsOneWidget);
    expect(find.text(l10n.whatsNewSectionNew), findsOneWidget);
    expect(find.text(l10n.whatsNewSectionImprovements), findsOneWidget);
    expect(find.text(l10n.whatsNewSectionFixes), findsOneWidget);
  });

  testWidgets('renders every bullet of the current release', (tester) async {
    final l10n = await pumpWhatsNew(tester);

    for (final section in currentWhatsNewRelease.sections) {
      for (final bullet in section.bullets) {
        final text = bullet(l10n);
        expect(text, isNotEmpty);
        expect(
          find.text(text),
          findsOneWidget,
          reason: 'release bullet not rendered: $text',
        );
      }
    }
  });

  testWidgets('advertises nothing that was removed or abandoned', (
    tester,
  ) async {
    final l10n = await pumpWhatsNew(tester);

    final copy = <String>[
      l10n.whatsNewTitle,
      l10n.whatsNewDescription,
      for (final section in currentWhatsNewRelease.sections) ...[
        section.title(l10n),
        for (final bullet in section.bullets) bullet(l10n),
      ],
    ].join('\n');

    for (final forbidden in forbiddenSubstrings) {
      expect(
        copy.toLowerCase().contains(forbidden.toLowerCase()),
        isFalse,
        reason: 'release notes must not mention "$forbidden"',
      );
    }
  });

  testWidgets('renders Arabic right-to-left', (tester) async {
    final l10n = await pumpWhatsNew(tester, locale: const Locale('ar'));

    expect(
      Directionality.of(tester.element(find.byType(WhatsNewPage))),
      TextDirection.rtl,
    );
    expect(find.text(l10n.whatsNewSectionNew), findsOneWidget);
    expect(find.text(l10n.whatsNew020Fix1), findsOneWidget);
    expect(find.text('PocketClaw 0.2.0'), findsOneWidget);
  });

  testWidgets('mirrors the bullet marker between LTR and RTL', (tester) async {
    Future<({double marker, Rect text})> geometryFor(Locale locale) async {
      final l10n = await pumpWhatsNew(tester, locale: locale);
      final marker = tester
          .getCenter(find.byKey(const Key('whatsNewBulletMarker')).first)
          .dx;
      final text = tester.getRect(find.text(l10n.whatsNew020New1));
      return (marker: marker, text: text);
    }

    final ltr = await geometryFor(const Locale('en'));
    expect(
      ltr.marker,
      lessThan(ltr.text.left),
      reason: 'the marker leads the text in a left-to-right locale',
    );

    final rtl = await geometryFor(const Locale('ar'));
    expect(
      rtl.marker,
      greaterThan(rtl.text.right),
      reason: 'the marker must mirror to the right in Arabic',
    );
  });
}
