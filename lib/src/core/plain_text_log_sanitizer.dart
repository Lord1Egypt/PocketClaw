import 'dart:convert';

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
    r'\[(?:\?[0-9:;]+|[0-9][0-9:;]*)[ ]*[ABCDEFGHJKSTfmnsu]',
  );
  static final RegExp _otherEscape = RegExp(r'\x1B[ -/]*[@-~]');
  static final RegExp _routineRealtimeWebSocket = RegExp(
    r'(?:^| > )GET /(?:pocketclaw|pico)/ws (?:101|2[0-9]{2})(?:\s|$)',
  );
  static final RegExp _realtimeWebSocketRequest = RegExp(
    r'((?:^| > )[A-Z]+) /(?:pocketclaw|pico)/ws ([0-9]{3})(\s|$)',
  );
  static final RegExp _legacyGatewayStart = RegExp(
    r'Starting gateway process \([^\r\n)]*\)',
  );
  static final RegExp _realtimeLoggerComponent = RegExp(
    r'(^|[ \t])([A-Z]{3}) (?:pocketclaw|pico) ([^ \t]+:[0-9]+)([ \t]+>)',
    multiLine: true,
  );
  static final RegExp _realtimeLoggerCaller = RegExp(
    r'(^|[ \t])([A-Z]{3}) ([^ \t]+) (?:pocketclaw|pico)\.go:([0-9]+)([ \t]+>)',
    multiLine: true,
  );
  static final RegExp _routineTelegramGetUpdatesCall = RegExp(
    r'(?:^| > )Telegram API call: getUpdates(?:, with data:.*)?$',
  );
  static final RegExp _telegramSuccessfulNilError = RegExp(
    r'((?:^| > )API response [A-Za-z][A-Za-z0-9_]*: Ok: true, Err:) \[<nil>\]',
  );
  static final RegExp _routineEmptyGetUpdatesResponse = RegExp(
    r'(?:^| > )API response getUpdates: Ok: true, Err: none, Result: \[\](?:\s|$)',
  );
  static final RegExp _sensitiveSessionField = RegExp(
    r'(^|[ \t])(session_key|scope_key|route_main_session)=(?:"(?:\\.|[^"\\])*"|[^ \t\r\n]*)',
    multiLine: true,
  );
  static final RegExp _internalIdentityField = RegExp(
    r'(^|[ \t])(chat_id|inbound_chat_id|target_chat_id|sender_id|inbound_sender_id|user_id|session_id|connection_id|conn_id|runtime_id)=(?:"(?:\\.|[^"\\])*"|[^ \t\r\n]*)',
    multiLine: true,
  );
  static final RegExp _rawContentField = RegExp(
    r'(^|[ \t])(arguments|args|content|messages_json|payload|preview|prompt|reasoning|response|text|tools_json)=(?:"(?:\\.|[^"\\])*"|[^ \t\r\n]*)',
    multiLine: true,
  );
  static final RegExp _jsonCredentialField = RegExp(
    r'("(?:api[_-]?key|telegram[_-]?token|token|cookie|authorization|credential|secret|password|core[_-]?bearer)"\s*:\s*)"(?:\\.|[^"\\])*"',
    caseSensitive: false,
  );
  static final RegExp _credentialAssignment = RegExp(
    r'(^|[ \t])((?:api[_-]?key|telegram[_-]?token|token|cookie|authorization|credential|secret|password|core[_-]?bearer)=)(?:"(?:\\.|[^"\\])*"|[^ \t\r\n]*)',
    caseSensitive: false,
    multiLine: true,
  );
  static final RegExp _bearerCredential = RegExp(
    r'\bBearer[ \t]+[A-Za-z0-9._~+/=-]{8,}',
    caseSensitive: false,
  );
  static final RegExp _commonApiKey = RegExp(r'\bsk-[A-Za-z0-9_-]{12,}\b');
  static final RegExp _telegramApiResponse = RegExp(
    r'^API response ([A-Za-z][A-Za-z0-9_]*): Ok: (true|false), Err: \[([\s\S]*?)\](?:, Result: ([\s\S]*))?$',
  );
  static final RegExp _safeTelegramApiCall = RegExp(
    r'^Telegram API call: ([A-Za-z][A-Za-z0-9_]*)(?:, with data:.*)?$',
    caseSensitive: false,
  );
  static const List<String> _telegramUpdateTypes = <String>[
    'message',
    'edited_message',
    'channel_post',
    'edited_channel_post',
    'business_connection',
    'business_message',
    'edited_business_message',
    'deleted_business_messages',
    'message_reaction',
    'message_reaction_count',
    'inline_query',
    'chosen_inline_result',
    'callback_query',
    'shipping_query',
    'pre_checkout_query',
    'purchased_paid_media',
    'poll',
    'poll_answer',
    'my_chat_member',
    'chat_member',
    'chat_join_request',
    'chat_boost',
    'removed_chat_boost',
  ];

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
      _realtimeLoggerComponent,
      (match) =>
          '${match.group(1)}${match.group(2)} realtime '
          '${match.group(3)}${match.group(4)}',
    );
    result = result.replaceAllMapped(
      _realtimeLoggerCaller,
      (match) =>
          '${match.group(1)}${match.group(2)} ${match.group(3)} '
          'realtime.go:${match.group(4)}${match.group(5)}',
    );
    result = _normalizeTelegramMessageInLine(result);
    if (result.isEmpty) return '';
    result = result.replaceAllMapped(
      _telegramSuccessfulNilError,
      (match) => '${match.group(1)} none',
    );
    result = _normalizePrivateStructuredFields(result);
    result = _normalizeRealtimeStructuredFields(result);
    if (_routineTelegramGetUpdatesCall.hasMatch(result) ||
        _routineEmptyGetUpdatesResponse.hasMatch(result)) {
      return '';
    }
    if (_routineRealtimeWebSocket.hasMatch(result)) return '';
    result = result.replaceAllMapped(
      _realtimeWebSocketRequest,
      (match) =>
          '${match.group(1)} /internal realtime connection '
          '${match.group(2)}${match.group(3)}',
    );
    return result;
  }

  static String _normalizeTelegramMessageInLine(String input) {
    final separator = input.indexOf(' > ');
    final messageStart = separator < 0 ? 0 : separator + ' > '.length;
    final prefix = input.substring(0, messageStart);
    final message = input.substring(messageStart);

    final call = _safeTelegramApiCall.firstMatch(message);
    if (call != null) {
      final operation = call.group(1)!;
      if (operation.toLowerCase() == 'getupdates') return '';
      return '${prefix}Telegram API call: $operation';
    }

    final response = _telegramApiResponse.firstMatch(message);
    if (response == null) return input;
    final operation = response.group(1)!;
    final ok = response.group(2)!;
    final error = response.group(3)!;
    final rawResult = response.group(4);
    if (ok == 'false') {
      final errorCode = RegExp(
        r'^([0-9]+)(?:\s|$)',
      ).firstMatch(error)?.group(1);
      return '${prefix}Telegram API failed operation=$operation ok=false'
          '${errorCode == null ? '' : ' error_code=$errorCode'}';
    }
    if (operation.toLowerCase() != 'getupdates') {
      return '${prefix}Telegram API completed operation=$operation ok=true';
    }
    if (rawResult == null) {
      return '${prefix}Telegram API response operation=getUpdates '
          'ok=true result=malformed';
    }

    try {
      final decoded = jsonDecode(rawResult);
      if (decoded is! List<Object?>) throw const FormatException();
      if (decoded.isEmpty) return '';
      final types = <String>{};
      for (final update in decoded) {
        if (update is! Map<String, Object?>) {
          types.add('unknown');
          continue;
        }
        String? updateType;
        for (final candidate in _telegramUpdateTypes) {
          if (update[candidate] != null) {
            updateType = candidate;
            break;
          }
        }
        types.add(updateType ?? 'unknown');
      }
      final sortedTypes = types.toList()..sort();
      final typeField = sortedTypes.length == 1 ? 'type' : 'types';
      return '${prefix}Telegram update received updates=${decoded.length} '
          '$typeField=${sortedTypes.join(',')}';
    } on FormatException {
      return '${prefix}Telegram API response operation=getUpdates '
          'ok=true result=malformed';
    }
  }

  static String _normalizePrivateStructuredFields(String input) {
    var result = input.replaceAllMapped(
      _jsonCredentialField,
      (match) => '${match.group(1)}"<redacted>"',
    );
    result = result.replaceAllMapped(
      _credentialAssignment,
      (match) => '${match.group(1)}${match.group(2)}<redacted>',
    );
    result = result
        .replaceAll(_bearerCredential, 'Bearer <redacted>')
        .replaceAll(_commonApiKey, '<redacted>');
    result = result.replaceAllMapped(
      _sensitiveSessionField,
      (match) => '${match.group(1)}${match.group(2)}=<redacted>',
    );
    result = result.replaceAllMapped(
      _internalIdentityField,
      (match) => '${match.group(1)}${match.group(2)}=<internal>',
    );
    return result.replaceAllMapped(
      _rawContentField,
      (match) => '${match.group(1)}${match.group(2)}=<redacted>',
    );
  }

  static String _normalizeRealtimeStructuredFields(String input) {
    const channelFields = <String>[
      'channel',
      'inbound_channel',
      'route_channel',
      'scope_channel',
      'target_channel',
    ];
    var result = input;
    for (final field in channelFields) {
      result = result.replaceAllMapped(
        RegExp('(^|[ \\t])$field=pico(?=\$|[ \\t\\r\\n])', multiLine: true),
        (match) => '${match.group(1)}$field=pocketclaw',
      );
    }
    return result;
  }
}
