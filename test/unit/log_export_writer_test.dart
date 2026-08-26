import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/log_export_writer.dart';
import 'package:pocketclaw/src/core/plain_text_log_sanitizer.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('pocketclaw.test/log-export');

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test(
    'Android Export Logs transport writes valid UTF-8 plain Unicode',
    () async {
      const raw =
          '\x1b[31mمدة 53.616µs ✅\x1b[0m\n'
          '123.4ms\nEnglish + عربي + 😊 + “Unicode”';
      const expected =
          'مدة 53.616µs ✅\n'
          '123.4ms\nEnglish + عربي + 😊 + “Unicode”';
      final sanitized = PlainTextLogSanitizer.sanitize(raw);

      MethodCall? capturedCall;
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, (call) async {
            capturedCall = call;
            return 'content://downloads/pocketclaw_logs_test.txt';
          });

      final result = await LogExportWriter.saveToAndroidDownloads(
        filename: 'pocketclaw_logs_test.txt',
        content: sanitized,
        channel: channel,
      );

      expect(result, 'content://downloads/pocketclaw_logs_test.txt');
      expect(capturedCall?.method, 'saveToDownloads');
      final arguments = Map<Object?, Object?>.from(
        capturedCall?.arguments as Map,
      );
      expect(arguments['filename'], 'pocketclaw_logs_test.txt');
      final bytes = arguments['bytes'] as Uint8List;
      final decoded = utf8.decode(bytes, allowMalformed: false);
      expect(decoded, expected);
      expect(decoded, contains('53.616µs'));
      expect(decoded, contains('عربي'));
      expect(decoded, contains('😊'));
      expect(decoded, isNot(contains('\uFFFD')));
      expect(decoded, isNot(contains('\x1b')));
    },
  );
}
