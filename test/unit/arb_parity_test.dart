import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Every locale must expose the same keys. Adding a string to one ARB and
/// forgetting the other eleven is the way a translated app quietly falls back
/// to English, and gen-l10n does not fail the build over it.
void main() {
  test('all ARB files declare the same message keys', () {
    final files =
        Directory('lib/l10n')
            .listSync()
            .whereType<File>()
            .where((file) => file.path.endsWith('.arb'))
            .toList()
          ..sort((a, b) => a.path.compareTo(b.path));

    expect(files.length, 12, reason: 'expected 12 supported locales');

    Set<String> keysOf(File file) {
      final decoded =
          jsonDecode(file.readAsStringSync()) as Map<String, dynamic>;
      return decoded.keys.where((key) => !key.startsWith('@')).toSet();
    }

    final reference = keysOf(
      files.firstWhere((file) => file.path.endsWith('app_en.arb')),
    );

    for (final file in files) {
      final keys = keysOf(file);
      expect(
        keys.difference(reference),
        isEmpty,
        reason: '${file.path} declares keys English does not',
      );
      expect(
        reference.difference(keys),
        isEmpty,
        reason: '${file.path} is missing keys English declares',
      );
    }
  });

  test(
    'the context memory strings are translated, not copied from English',
    () {
      final english =
          jsonDecode(File('lib/l10n/app_en.arb').readAsStringSync())
              as Map<String, dynamic>;

      const translatedKeys = <String>[
        'contextMemoryTitle',
        'contextMemoryDescription',
        'contextMemoryHelp',
        'contextMemoryRecommended',
        'contextMemoryCustom',
      ];

      for (final locale in <String>[
        'ar',
        'de',
        'es',
        'fr',
        'hi',
        'ja',
        'ko',
        'ru',
        'zh',
      ]) {
        final decoded =
            jsonDecode(File('lib/l10n/app_$locale.arb').readAsStringSync())
                as Map<String, dynamic>;
        for (final key in translatedKeys) {
          expect(decoded[key], isNotNull, reason: '$locale is missing $key');
          expect(
            decoded[key],
            isNot(equals(english[key])),
            reason: '$locale copied the English $key instead of translating it',
          );
        }
      }
    },
  );
}
