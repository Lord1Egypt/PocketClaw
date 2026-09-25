import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// PC-DEF-087, seen on the phone: a left-to-right path such as
/// `Download/pocketclaw` inside Arabic text is reordered when the line wraps,
/// and the slash lands at the wrong end. Every such path in the right-to-left
/// bundle must sit between U+2066 (LEFT-TO-RIGHT ISOLATE) and U+2069.
void main() {
  test('Latin paths in Arabic strings are bidi-isolated', () {
    final bundle =
        jsonDecode(File('lib/l10n/app_ar.arb').readAsStringSync())
            as Map<String, dynamic>;
    final path = RegExp(r'[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)+');
    final offenders = <String>[];
    bundle.forEach((key, value) {
      if (key.startsWith('@') || value is! String) return;
      for (final match in path.allMatches(value)) {
        final before = match.start > 0 ? value[match.start - 1] : '';
        final after = match.end < value.length ? value[match.end] : '';
        final isUrl = value.substring(0, match.start).endsWith('://');
        if (!isUrl && (before != '⁦' || after != '⁩')) {
          offenders.add('$key: ${match.group(0)}');
        }
      }
    });
    expect(offenders, isEmpty);
  });
}
