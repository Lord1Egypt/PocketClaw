import 'dart:convert';
import 'dart:io';

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

  test('preserves valid multibyte text surrounding ANSI sequences', () {
    const raw =
        '\x1b[31mمدة 53.616µs ✅\x1b[0m — '
        '\x1b]0;discarded title\x07English عربي 😊 «آمن»';
    const expected = 'مدة 53.616µs ✅ — English عربي 😊 «آمن»';

    final result = clean(raw);

    expect(result, expected);
    expect(result, contains('53.616µs'));
    expect(result, isNot(contains('\uFFFD')));
    expect(utf8.decode(utf8.encode(result), allowMalformed: false), result);
  });

  test('preserves plain millisecond durations', () {
    expect(clean('123.4ms'), '123.4ms');
  });

  test('preserves angle brackets as ordinary plain text', () {
    for (final value in <String>[
      'Err: <nil>',
      'value=<none>',
      '1 < 2',
      '2 > 1',
      '<tag>',
      'Arabic <نص>',
      'emoji 🚀 <nil>',
    ]) {
      expect(clean(value), value);
      expect(clean(value), isNot(contains('\uFFFD')));
    }
    expect(
      clean('API response getMe: Ok: true, Err: [<nil>], Result: {}'),
      'Telegram API completed operation=getMe ok=true',
    );
  });

  test('suppresses only successful empty Telegram long polling', () {
    expect(clean('DBG telego bot.go:247 > Telegram API call: getUpdates'), '');
    expect(
      clean(
        'DBG telego bot.go:173 > API response getUpdates: '
        'Ok: true, Err: [<nil>], Result: []',
      ),
      '',
    );
    expect(
      clean(
        'DBG telego bot.go:173 > API response getUpdates: '
        'Ok: true, Err: [<nil>], Result: [{"update_id":42}]',
      ),
      contains('Telegram update received updates=1 type=unknown'),
    );
    expect(
      clean(
        'ERR telego bot.go:170 > Execution error getUpdates: '
        'context deadline exceeded',
      ),
      contains('context deadline exceeded'),
    );
  });

  test('redacts credentials in structured and free-form log lines', () {
    const raw =
        '{"event":"gateway.start.failed","token":"telegram-child-token",'
        '"reason":"Bearer abcdefghijklmnop",'
        '"error":"provider rejected sk-abcdefghijklmnop"}';
    final result = clean(raw);
    final decoded = jsonDecode(result) as Map<String, Object?>;

    expect(decoded['token'], '<redacted>');
    expect(decoded['reason'], 'Bearer <redacted>');
    expect(decoded['error'], 'provider rejected <redacted>');
    expect(result, isNot(contains('telegram-child-token')));
    expect(result, isNot(contains('abcdefghijklmnop')));
  });

  test('redacts credential assignments without altering other fields', () {
    final result = clean('INF core > start api_key=secret-value port=18800');

    expect(result, contains('api_key=<redacted>'));
    expect(result, contains('port=18800'));
    expect(result, isNot(contains('secret-value')));
  });

  test('matches the shared user-visible log contract', () async {
    final raw = await File(
      'test/fixtures/user_visible_log_contract.json',
    ).readAsString();
    final cases = (jsonDecode(raw) as List<Object?>)
        .cast<Map<String, Object?>>();

    for (final fixture in cases) {
      final input = fixture['input']! as String;
      final expected = fixture['expected']! as String;
      final result = clean(input);

      expect(result, expected, reason: fixture['name']! as String);
      expect(
        utf8.decode(utf8.encode(result), allowMalformed: false),
        result,
        reason: fixture['name']! as String,
      );
      if (!input.contains('\uFFFD')) {
        expect(
          result,
          isNot(contains('\uFFFD')),
          reason: fixture['name']! as String,
        );
      }
    }
  });
}
