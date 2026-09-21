import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/whats_new_page.dart';
import 'package:pocketclaw/src/whats_new/whats_new_release.dart';

/// Words this release must never advertise. Some name work that was removed,
/// some name work that was abandoned, and some are internal build detail no
/// user should read in release notes.
///
/// WHATSAPP-GUARD-ENFORCEMENT-DATA — this list is the mechanism of the
/// prohibition, not a surface that breaks it. The Core-side guard that scans
/// for a returning WhatsApp surface reads this marker and skips this file;
/// without it, the enforcement list flagged itself. Do not add this marker to a
/// file that is not itself enforcing a prohibition.
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

  // Written against currentWhatsNewRelease rather than a pinned version and a
  // fixed set of headings: a patch release legitimately ships one section, and
  // a test that has to be edited every release stops testing the page.
  testWidgets('renders the current release header and its sections', (
    tester,
  ) async {
    final l10n = await pumpWhatsNew(tester);

    expect(
      find.text('PocketClaw ${currentWhatsNewRelease.version}'),
      findsOneWidget,
    );
    for (final section in currentWhatsNewRelease.sections) {
      expect(find.text(section.title(l10n)), findsOneWidget);
    }
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
    final first = currentWhatsNewRelease.sections.first;
    expect(find.text(first.title(l10n)), findsOneWidget);
    expect(find.text(first.bullets.first(l10n)), findsOneWidget);
    expect(
      find.text('PocketClaw ${currentWhatsNewRelease.version}'),
      findsOneWidget,
    );
  });

  testWidgets('mirrors the bullet marker between LTR and RTL', (tester) async {
    Future<({double marker, Rect text})> geometryFor(Locale locale) async {
      final l10n = await pumpWhatsNew(tester, locale: locale);
      final marker = tester
          .getCenter(find.byKey(const Key('whatsNewBulletMarker')).first)
          .dx;
      final text = tester.getRect(
        find.text(currentWhatsNewRelease.sections.first.bullets.first(l10n)),
      );
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
