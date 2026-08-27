/// Converts terminal output into safe plain text for PocketClaw's Logs UI.
///
/// This intentionally removes terminal control protocols, not Unicode. Arabic,
/// emoji, box-drawing characters, and other printable text remain untouched.
abstract final class PlainTextLogSanitizer {
  static final RegExp _osc = RegExp(
    r'(?:\x1B\]|\u009D)[\s\S]*?(?:\x07|\x1B\\|\u009C)',
  );
  static final RegExp _unterminatedOsc = RegExp(
    r'(?:\x1B\]|\u009D)[^\n]*$',
    multiLine: true,
  );
  static final RegExp _stringControl = RegExp(
    r'(?:\x1B[P\^_X]|[\u0090\u0098\u009E\u009F])[\s\S]*?(?:\x1B\\|\u009C)',
  );
  static final RegExp _unterminatedStringControl = RegExp(
    r'(?:\x1B[P\^_X]|[\u0090\u0098\u009E\u009F])[^\n]*$',
    multiLine: true,
  );
  static final RegExp _csi = RegExp(r'(?:\x1B\[|\u009B)[0-?]*[ -/]*[@-~]');
  static final RegExp _orphanedCsi = RegExp(
    r'\[(?:\??[0-9:;<=>]+)[0-9:;<=>? ]*[ABCDEFGHJKSTfmnsu]',
  );
  static final RegExp _otherEscape = RegExp(r'\x1B[ -/]*[@-~]');
  static final RegExp _routinePicoWebSocket = RegExp(
    r'(?:^| > )GET /pico/ws (?:101|2[0-9]{2})(?:\s|$)',
  );
  static final RegExp _picoWebSocketRequest = RegExp(
    r'((?:^| > )[A-Z]+) /pico/ws ([0-9]{3})(\s|$)',
  );
  static final RegExp _legacyGatewayStart = RegExp(
    r'Starting gateway process \([^\r\n)]*\)',
  );
  static final RegExp _picoLoggerComponent = RegExp(
    r'(^|[ \t])([A-Z]{3}) pico ([^ \t]+:[0-9]+)([ \t]+>)',
    multiLine: true,
  );
  static final RegExp _picoLoggerCaller = RegExp(
    r'(^|[ \t])([A-Z]{3}) ([^ \t]+) pico\.go:([0-9]+)([ \t]+>)',
    multiLine: true,
  );

  static String sanitize(String input) {
    if (input.isEmpty) return input;

    var text = input
        .replaceAll('\r\n', '\n')
        .replaceAll(_osc, '')
        .replaceAll(_unterminatedOsc, '')
        .replaceAll(_stringControl, '')
        .replaceAll(_unterminatedStringControl, '')
        .replaceAll(_csi, '')
        // Some bridges discard ESC itself but leave the numeric CSI suffix.
        .replaceAll(_orphanedCsi, '')
        .replaceAll(_otherEscape, '');

    final output = StringBuffer();
    final currentLine = <int>[];

    void flushLine({required bool newline}) {
      output.write(String.fromCharCodes(currentLine));
      currentLine.clear();
      if (newline) output.write('\n');
    }

    for (final rune in text.runes) {
      if (rune == 0x0A) {
        flushLine(newline: true);
      } else if (rune == 0x0D) {
        // A terminal would return to column zero and overwrite progress text.
        currentLine.clear();
      } else if (rune == 0x08) {
        if (currentLine.isNotEmpty) currentLine.removeLast();
      } else if (rune == 0x09) {
        currentLine.add(rune);
      } else if (rune < 0x20 || (rune >= 0x7F && rune <= 0x9F)) {
        // C0/C1 controls and DEL have no representation in a plain-text log.
        continue;
      } else {
        currentLine.add(rune);
      }
    }
    flushLine(newline: false);

    var result = output.toString().replaceAll(
      _legacyGatewayStart,
      'Starting gateway process',
    );
    result = result.replaceAllMapped(
      _picoLoggerComponent,
      (match) =>
          '${match.group(1)}${match.group(2)} realtime '
          '${match.group(3)}${match.group(4)}',
    );
    result = result.replaceAllMapped(
      _picoLoggerCaller,
      (match) =>
          '${match.group(1)}${match.group(2)} ${match.group(3)} '
          'realtime.go:${match.group(4)}${match.group(5)}',
    );
    if (_routinePicoWebSocket.hasMatch(result)) return '';
    result = result.replaceAllMapped(
      _picoWebSocketRequest,
      (match) =>
          '${match.group(1)} /internal realtime connection '
          '${match.group(2)}${match.group(3)}',
    );
    return result;
  }
}
