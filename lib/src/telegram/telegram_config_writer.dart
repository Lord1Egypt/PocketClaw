import 'dart:convert';

import '../core/picoclaw_channel.dart';
import 'telegram_onboarding_models.dart';

/// Writes a paired bot's credentials into Core's existing Telegram channel
/// configuration.
///
/// PocketClaw does not gain a second Telegram runtime from managed-bot
/// onboarding. The bot that gets created is configured through exactly the
/// same `channel_list.telegram` entry that manual setup writes, so everything
/// downstream — the channel implementation, streaming, MarkdownV2, the agent
/// identity behind `/start` — is unchanged.
class TelegramConfigWriter {
  const TelegramConfigWriter({
    Future<String> Function()? readConfig,
    Future<bool> Function(String)? writeConfig,
  })  : _readConfig = readConfig ?? PicoClawChannel.getConfig,
        _writeConfig = writeConfig ?? PicoClawChannel.saveConfig;

  final Future<String> Function() _readConfig;
  final Future<bool> Function(String) _writeConfig;

  /// Reads Core's configuration, applies [credentials], and writes it back.
  ///
  /// Throws [TelegramOnboardingException] with
  /// [TelegramOnboardingErrorKind.configurationFailed] if the configuration
  /// cannot be read, parsed, or saved. It never falls through to a partially
  /// applied state: the merge happens in memory and is written in one call.
  Future<void> apply(TelegramBotCredentials credentials) async {
    final String raw;
    try {
      raw = await _readConfig();
    } catch (_) {
      throw const TelegramOnboardingException(
        TelegramOnboardingErrorKind.configurationFailed,
        'could not read the Core configuration',
      );
    }

    final String merged;
    try {
      merged = applyToConfigJson(raw, credentials);
    } on FormatException {
      throw const TelegramOnboardingException(
        TelegramOnboardingErrorKind.configurationFailed,
        'the Core configuration is not valid JSON',
      );
    }

    final bool saved;
    try {
      saved = await _writeConfig(merged);
    } catch (_) {
      throw const TelegramOnboardingException(
        TelegramOnboardingErrorKind.configurationFailed,
        'could not save the Core configuration',
      );
    }
    if (!saved) {
      throw const TelegramOnboardingException(
        TelegramOnboardingErrorKind.configurationFailed,
        'the Core configuration was rejected',
      );
    }
  }

  /// The pure transformation, exposed so it can be tested without any I/O.
  ///
  /// Merges rather than replaces: an existing `channel_list.telegram` entry
  /// keeps its streaming, MarkdownV2, proxy, and base-URL settings, and every
  /// other channel is left untouched.
  static String applyToConfigJson(
    String rawConfig,
    TelegramBotCredentials credentials,
  ) {
    final Map<String, dynamic> config;
    if (rawConfig.trim().isEmpty) {
      config = <String, dynamic>{};
    } else {
      final decoded = jsonDecode(rawConfig);
      if (decoded is! Map) {
        throw const FormatException('config root is not an object');
      }
      config = Map<String, dynamic>.from(decoded);
    }

    final channels = Map<String, dynamic>.from(
      (config['channel_list'] as Map?) ?? const <String, dynamic>{},
    );
    final telegram = Map<String, dynamic>.from(
      (channels['telegram'] as Map?) ?? const <String, dynamic>{},
    );
    final settings = Map<String, dynamic>.from(
      (telegram['settings'] as Map?) ?? const <String, dynamic>{},
    );

    settings['token'] = credentials.token;

    telegram['type'] = 'telegram';
    telegram['enabled'] = true;
    telegram['settings'] = settings;
    // The pairing tells us exactly who created the bot, so a freshly paired
    // bot answers only its owner instead of anyone who finds it. This is the
    // one thing managed-bot onboarding knows that manual setup cannot.
    telegram['allow_from'] = <String>['${credentials.ownerUserId}'];

    channels['telegram'] = telegram;
    config['channel_list'] = channels;

    return const JsonEncoder.withIndent('  ').convert(config);
  }
}
