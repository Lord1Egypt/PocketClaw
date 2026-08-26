import 'dart:convert';

import '../core/picoclaw_channel.dart';

/// Telegram's connection state, as persisted by Core.
///
/// There is deliberately no stored "connected" boolean anywhere. Both surfaces
/// — the native Settings card and Channels → Telegram in the embedded console
/// — derive this from the same `channel_list.telegram` entry that Core itself
/// reads, so the two cannot drift apart.
class TelegramConnectionStatus {
  const TelegramConnectionStatus({
    required this.configured,
    required this.ownerRestricted,
    this.botUsername,
  });

  static const TelegramConnectionStatus disconnected =
      TelegramConnectionStatus(configured: false, ownerRestricted: false);

  /// Whether a bot token is persisted. This is the same condition Core's own
  /// `detectConfiguredSecrets` reports to the web console as `token`.
  final bool configured;

  /// Whether `allow_from` restricts the bot to specific Telegram users.
  final bool ownerRestricted;

  /// The paired bot's public `@username`, when this device paired it.
  ///
  /// Core does not store the handle, so a manually configured bot — or one
  /// paired on another device — is connected but has no handle to show. That
  /// is reported honestly rather than guessed at.
  final String? botUsername;

  String? get chatUrl =>
      botUsername == null ? null : 'https://t.me/$botUsername';

  @override
  String toString() => 'TelegramConnectionStatus(configured: $configured, '
      'ownerRestricted: $ownerRestricted, botUsername: $botUsername)';
}

abstract final class TelegramConnectionReader {
  /// Reads the persisted configuration and reports Telegram's state.
  ///
  /// Fails closed: an unreadable or malformed configuration reports
  /// disconnected rather than claiming a connection that may not exist.
  static Future<TelegramConnectionStatus> read({
    Future<String> Function()? readConfig,
    Future<String?> Function()? readBotUsername,
  }) async {
    final loadConfig = readConfig ?? PicoClawChannel.getConfig;
    String raw;
    try {
      raw = await loadConfig();
    } catch (_) {
      return TelegramConnectionStatus.disconnected;
    }

    final username = readBotUsername == null ? null : await readBotUsername();
    return parse(raw, botUsername: username);
  }

  /// The pure transformation, so the state both surfaces show can be asserted
  /// against real configuration shapes without any I/O.
  static TelegramConnectionStatus parse(
    String rawConfig, {
    String? botUsername,
  }) {
    Object? decoded;
    try {
      decoded = jsonDecode(rawConfig);
    } catch (_) {
      return TelegramConnectionStatus.disconnected;
    }
    if (decoded is! Map) return TelegramConnectionStatus.disconnected;

    final channels = decoded['channel_list'];
    if (channels is! Map) return TelegramConnectionStatus.disconnected;
    final telegram = channels['telegram'];
    if (telegram is! Map) return TelegramConnectionStatus.disconnected;

    final settings = telegram['settings'];
    final token = settings is Map ? settings['token'] : null;
    final configured = token is String && token.trim().isNotEmpty;
    if (!configured) return TelegramConnectionStatus.disconnected;

    final allowFrom = telegram['allow_from'];
    final ownerRestricted = allowFrom is List &&
        allowFrom.any((e) => e != null && e.toString().trim().isNotEmpty);

    final handle = botUsername?.trim();
    return TelegramConnectionStatus(
      configured: true,
      ownerRestricted: ownerRestricted,
      botUsername: handle == null || handle.isEmpty ? null : handle,
    );
  }
}
