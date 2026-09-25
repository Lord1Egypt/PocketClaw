import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/whats_new_page.dart';
import 'package:pocketclaw/src/whats_new/whats_new_release.dart';

/// What's New is a release history. When 0.2.2 was added it replaced the
/// single "current release" and 0.2.1 and 0.2.0 vanished from the screen,
/// although their definitions and translations were still in the bundle.
void main() {
  List<int> parts(String version) => version.split('.').map(int.parse).toList();

  int compare(String a, String b) {
    final x = parts(a), y = parts(b);
    for (var i = 0; i < 3; i++) {
      if (x[i] != y[i]) return x[i].compareTo(y[i]);
    }
    return 0;
  }

  group('the history data', () {
    test('holds 0.2.3, 0.2.2, 0.2.1 and 0.2.0, newest first', () {
      final versions = whatsNewHistory.map((r) => r.version).toList();
      expect(versions.take(4), ['0.2.3', '0.2.2', '0.2.1', '0.2.0']);
      for (var i = 1; i < versions.length; i++) {
        expect(compare(versions[i - 1], versions[i]), greaterThan(0));
      }
      expect(currentWhatsNewRelease, same(whatsNewHistory.first));
    });

    test('every release defined in the source is listed in the history', () {
      // Adding a release means defining it and putting it at the front of the
      // list. A defined release that is not listed would silently disappear
      // from the screen, which is how the history was lost.
      final source = File(
        'lib/src/whats_new/whats_new_release.dart',
      ).readAsStringSync();
      final defined = RegExp(
        r'final WhatsNewRelease (whatsNewRelease\d+) =',
      ).allMatches(source).map((m) => m.group(1)!).toSet();
      final listed = RegExp(r'whatsNewHistory = \[([^\]]*)\]')
          .firstMatch(source)!
          .group(1)!
          .split(',')
          .map((s) => s.trim())
          .where((s) => s.isNotEmpty)
          .toSet();
      expect(defined, isNotEmpty);
      expect(listed, defined);
    });

    test(
      'the notes are bundled: nothing in What\'s New reaches the network',
      () {
        for (final path in const [
          'lib/src/whats_new/whats_new_release.dart',
          'lib/src/ui/whats_new_page.dart',
        ]) {
          final source = File(path).readAsStringSync();
          expect(source, isNot(contains('package:http')), reason: path);
          expect(source, isNot(contains('HttpClient')), reason: path);
          expect(source, isNot(contains("import 'dart:io'")), reason: path);
        }
      },
    );
  });

  group('every locale carries the whole history', () {
    for (final locale in AppLocalizations.supportedLocales) {
      test(locale.toLanguageTag(), () async {
        final l10n = await AppLocalizations.delegate.load(locale);
        final english = await AppLocalizations.delegate.load(
          const Locale('en'),
        );
        final nonLatin = const {
          'ar',
          'hi',
          'ja',
          'ko',
          'ru',
          'zh',
        }.contains(locale.languageCode);
        for (final release in whatsNewHistory) {
          for (final section in release.sections) {
            expect(section.title(l10n).trim(), isNotEmpty);
            for (final bullet in section.bullets) {
              final text = bullet(l10n);
              expect(text.trim(), isNotEmpty, reason: release.version);
              if (nonLatin) {
                expect(
                  text,
                  isNot(bullet(english)),
                  reason: '${release.version} is English in $locale',
                );
              }
            }
          }
        }
      });
    }
  });

  group('the page', () {
    Future<AppLocalizations> pump(
      WidgetTester tester, {
      Locale locale = const Locale('en'),
      List<WhatsNewRelease>? releases,
    }) async {
      tester.view.physicalSize = const Size(1080, 2340);
      tester.view.devicePixelRatio = 2.625;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(
        MaterialApp(
          locale: locale,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: WhatsNewPage(releases: releases),
        ),
      );
      await tester.pumpAndSettle();
      return AppLocalizations.of(tester.element(find.byType(WhatsNewPage)))!;
    }

    Future<void> openRelease(WidgetTester tester, String version) async {
      final title = find.text('PocketClaw $version');
      await tester.scrollUntilVisible(title, 300);
      await tester.tap(title);
      await tester.pumpAndSettle();
    }

    testWidgets('shows the current release open and the earlier ones listed', (
      tester,
    ) async {
      final l10n = await pump(tester);
      expect(find.text('PocketClaw 0.2.3'), findsOneWidget);
      expect(
        find.text(whatsNewRelease023.sections.first.bullets.first(l10n)),
        findsOneWidget,
      );
      for (final version in const ['0.2.2', '0.2.1', '0.2.0']) {
        await tester.scrollUntilVisible(find.text('PocketClaw $version'), 300);
        expect(find.text('PocketClaw $version'), findsOneWidget);
      }
    });

    testWidgets('an earlier release opens in place with its real notes', (
      tester,
    ) async {
      final l10n = await pump(tester);
      expect(find.text(l10n.whatsNew022Fix3), findsNothing);
      await openRelease(tester, '0.2.2');
      await tester.scrollUntilVisible(find.text(l10n.whatsNew022Fix3), 300);
      expect(find.text(l10n.whatsNew022Fix3), findsOneWidget);
      expect(find.text(l10n.whatsNew021Fix1), findsNothing);
      await openRelease(tester, '0.2.1');
      expect(find.text(l10n.whatsNew021Fix1), findsOneWidget);
      await openRelease(tester, '0.2.0');
      await tester.scrollUntilVisible(find.text(l10n.whatsNew020Fix2), 300);
      expect(find.text(l10n.whatsNew020Fix2), findsOneWidget);
    });

    testWidgets('a newer release is added above, and nothing older is lost', (
      tester,
    ) async {
      final next = WhatsNewRelease(
        version: '0.2.4',
        sections: whatsNewRelease021.sections,
      );
      await pump(tester, releases: [next, ...whatsNewHistory]);
      expect(find.text('PocketClaw 0.2.4'), findsOneWidget);
      for (final version in const ['0.2.3', '0.2.2', '0.2.1', '0.2.0']) {
        await tester.scrollUntilVisible(find.text('PocketClaw $version'), 300);
        expect(find.text('PocketClaw $version'), findsOneWidget);
      }
    });

    testWidgets('Arabic lays the history out right to left without overflow', (
      tester,
    ) async {
      final l10n = await pump(tester, locale: const Locale('ar'));
      expect(
        Directionality.of(tester.element(find.text('PocketClaw 0.2.3'))),
        TextDirection.rtl,
      );
      await openRelease(tester, '0.2.2');
      await openRelease(tester, '0.2.1');
      expect(find.text(l10n.whatsNew021Fix1), findsOneWidget);
      await openRelease(tester, '0.2.0');
      expect(tester.takeException(), isNull);
    });
  });
}
