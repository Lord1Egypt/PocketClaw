import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/plain_text_log_sanitizer.dart';

void main() {
  String clean(String value) => PlainTextLogSanitizer.sanitize(value);

  test('strips RGB, 256-color, style, reset, and nested SGR', () {
    const raw =
        '\x1b[38;2;213;70;70mERROR '
        '\x1b[38;5;185m\x1b[1mBold\x1b[4m under\x1b[0m';
    expect(clean(raw), 'ERROR Bold under');
  });

  test('strips OSC and terminal string protocols', () {
    const raw =
        'before\x1b]0;secret title\x07after '
        '\x1b]8;;https://example.invalid\x1b\\link\x1b]8;;\x1b\\ '
        '\x1bPprivate payload\x1b\\done';
    expect(clean(raw), 'beforeafter link done');
  });

  test('normalizes carriage return and backspace terminal updates', () {
    expect(clean('10%\r50%\rDone'), 'Done');
    expect(clean('abc\b\bd'), 'ad');
    expect(clean('first\r\nsecond'), 'first\nsecond');
  });

  test('strips cursor, erase line, erase screen, and multiple controls', () {
    const raw = '\x1b[2J\x1b[HStart\x1b[3C Middle\x1b[K\x1b[1A End\x1b[0m';
    expect(clean(raw), 'Start Middle End');
  });

  test('strips C1 CSI, controls, and orphaned numeric CSI fragments', () {
    const raw = '\u009b31mRed\u009b0m\u0000 [38;2;1;2;3mBlue[0m';
    final result = clean(raw);
    expect(result, 'Red Blue');
    expect(result, isNot(contains('\x1b')));
    expect(result, isNot(contains('[38;2;')));
    expect(result.runes.any((rune) => rune < 0x20), isFalse);
  });

  test('preserves Arabic, emoji, ordinary Unicode, tabs, and newlines', () {
    const text = 'تسلم\t😊\nالعفو — café 中文 ✓';
    expect(clean(text), text);
  });
}
