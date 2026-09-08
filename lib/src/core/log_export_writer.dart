import 'dart:convert';

import 'package:flutter/services.dart';

/// Writes the Logs screen's already-sanitized plain text as a UTF-8 file.
abstract final class LogExportWriter {
  static const MethodChannel _channel = MethodChannel(
    'com.lord1egypt.pocketclaw/pocketclaw',
  );

  /// Uses UTF-8 rather than truncating Dart's UTF-16 code units to bytes.
  static Uint8List encode(String content) =>
      Uint8List.fromList(utf8.encode(content));

  /// Saves through the Android MediaStore bridge used by the Logs page.
  ///
  /// The optional channel lets the transport boundary be exercised in a host
  /// test without changing the production channel.
  static Future<String?> saveToAndroidDownloads({
    required String filename,
    required String content,
    MethodChannel channel = _channel,
  }) {
    return channel.invokeMethod<String>('saveToDownloads', {
      'filename': filename,
      'bytes': encode(content),
    });
  }
}
