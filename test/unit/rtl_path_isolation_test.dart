import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// PC-DEF-087, seen on the phone: a left-to-right path such as
/// `Download/pocketclaw` inside Arabic text broke at the slash and rendered as
/// `/Download … pocketclaw`. A path whose slashes are joined with U+2060 (WORD
/// JOINER) cannot break there, and between two Latin letters the slash then
/// resolves left-to-right. Bidi isolates (U+2066/U+2069) would also work, but
/// gen-l10n copies them into Dart as literal characters, which the analyzer
/// rejects as a Trojan Source risk.
void main() {
  test('Latin paths in Arabic strings cannot break at a slash', () {
    final bundle =
        jsonDecode(File('lib/l10n/app_ar.arb').readAsStringSync())
            as Map<String, dynamic>;
    final path = RegExp(r'[A-Za-z0-9_.-]+(?:\u2060?/\u2060?[A-Za-z0-9_.-]+)+');
    final offenders = <String>[];
    bundle.forEach((key, value) {
      if (key.startsWith('@') || value is! String) return;
      for (final match in path.allMatches(value)) {
        final text = match.group(0)!;
        final isUrl = value.substring(0, match.start).endsWith('://');
        final slashes = '/'.allMatches(text).length;
        final joined = '\u2060/\u2060'.allMatches(text).length;
        if (!isUrl && joined != slashes) offenders.add('$key: $text');
      }
    });
    expect(offenders, isEmpty);
  });
}
