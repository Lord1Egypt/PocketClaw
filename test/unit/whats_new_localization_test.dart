import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// The What's New copy is release-notes prose. An untranslated locale is a
/// visible defect here in a way a missing internal label is not, so every key
/// is checked in every bundle rather than only in the English template.
const List<String> whatsNewKeys = <String>[
  'whatsNewTitle',
  'whatsNewDescription',
  'whatsNewBadge',
  'whatsNewSectionNew',
  'whatsNewSectionImprovements',
  'whatsNewSectionFixes',
  'whatsNew020New1',
  'whatsNew020New2',
  'whatsNew020New3',
  'whatsNew020New4',
  'whatsNew020New5',
  'whatsNew020Improvement1',
  'whatsNew020Improvement2',
  'whatsNew020Improvement3',
  'whatsNew020Fix1',
  'whatsNew020Fix2',
];

const List<String> locales = <String>[
  'ar',
  'de',
  'en',
  'es',
  'fr',
  'hi',
  'id',
  'ja',
  'ko',
  'pt',
  'ru',
  'zh',
];

/// Latin script is expected in these locales; elsewhere a value identical to
/// English is a copied placeholder.
const List<String> latinScriptLocales = <String>['de', 'es', 'fr', 'id', 'pt'];

Map<String, dynamic> bundleFor(String locale) =>
    jsonDecode(File('lib/l10n/app_$locale.arb').readAsStringSync())
        as Map<String, dynamic>;

void main() {
  test('every locale declares every What\'s New key with real text', () {
    for (final locale in locales) {
      final bundle = bundleFor(locale);
      for (final key in whatsNewKeys) {
        final value = bundle[key];
        expect(value, isA<String>(), reason: '$locale is missing $key');
        expect(
          (value as String).trim(),
          isNotEmpty,
          reason: '$locale has an empty $key',
        );
      }
    }
  });

  test('no locale ships the English release prose untranslated', () {
    final english = bundleFor('en');

    // Product and tool names are intentionally identical everywhere; only
    // the sentences around them have to be translated.
    const sentenceKeys = <String>[
      'whatsNewTitle',
      'whatsNewDescription',
      'whatsNewSectionNew',
      'whatsNewSectionImprovements',
      'whatsNewSectionFixes',
      'whatsNew020New1',
      'whatsNew020New3',
      'whatsNew020New4',
      'whatsNew020New5',
      'whatsNew020Improvement1',
      'whatsNew020Improvement2',
      'whatsNew020Improvement3',
      'whatsNew020Fix1',
      'whatsNew020Fix2',
    ];

    for (final locale in locales.where((locale) => locale != 'en')) {
      final bundle = bundleFor(locale);
      for (final key in sentenceKeys) {
        expect(
          bundle[key],
          isNot(equals(english[key])),
          reason: '$locale copied the English $key instead of translating it',
        );
      }
    }
  });

  test('the non-Latin locales are actually written in their own script', () {
    final nonLatin = locales
        .where((locale) => locale != 'en')
        .where((locale) => !latinScriptLocales.contains(locale));

    for (final locale in nonLatin) {
      final bundle = bundleFor(locale);
      // The bundled tool names stay Latin; the surrounding prose must not.
      final prose = (bundle['whatsNewDescription'] as String);
      expect(
        RegExp(r'[^\x00-\x7F]').hasMatch(prose),
        isTrue,
        reason: '$locale whatsNewDescription reads as ASCII, not $locale',
      );
    }
  });

  test('the What\'s New keys take no placeholders in any locale', () {
    final placeholder = RegExp(r'\{[A-Za-z_][A-Za-z0-9_]*\}');

    for (final locale in locales) {
      final bundle = bundleFor(locale);
      for (final key in whatsNewKeys) {
        expect(
          placeholder.hasMatch(bundle[key] as String),
          isFalse,
          reason: '$locale.$key introduced a placeholder English does not have',
        );
        expect(
          bundle.containsKey('@$key'),
          isFalse,
          reason: '@$key declares metadata for a message that takes none',
        );
      }
    }
  });
}
