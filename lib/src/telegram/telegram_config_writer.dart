import '../core/pocketclaw_channel.dart';
import 'telegram_onboarding_models.dart';

typedef TelegramCredentialSink =
    Future<bool> Function(TelegramBotCredentials credentials);

/// Persists a paired bot through Core's authoritative configuration boundary.
///
/// Core splits public configuration into `config.json` and credentials into
/// `.security.yml`. Writing only the first file is incorrect: secure fields in
/// JSON are deliberately redacted and the security file wins when Core loads.
/// This writer therefore gives the credential to Core, which updates both via
/// its normal SaveConfig path. No token is ever returned to Flutter.
class TelegramConfigWriter {
  const TelegramConfigWriter({TelegramCredentialSink? writeCredentials})
    : _writeCredentials = writeCredentials;

  final TelegramCredentialSink? _writeCredentials;

  Future<void> apply(TelegramBotCredentials credentials) async {
    if (credentials.ownerUserId <= 0) {
      throw const TelegramOnboardingException(
        TelegramOnboardingErrorKind.configurationFailed,
        'a verified numeric Telegram owner is required',
      );
    }
    final sink = _writeCredentials ?? _writeThroughCore;
    final bool saved;
    try {
      saved = await sink(credentials);
    } catch (_) {
      throw const TelegramOnboardingException(
        TelegramOnboardingErrorKind.configurationFailed,
        'could not save the Core Telegram configuration',
      );
    }
    if (!saved) {
      throw const TelegramOnboardingException(
        TelegramOnboardingErrorKind.configurationFailed,
        'the Core Telegram configuration was rejected',
      );
    }
  }

  static Future<bool> _writeThroughCore(TelegramBotCredentials credentials) =>
      PocketClawChannel.configureTelegram(
        token: credentials.token,
        ownerUserId: credentials.ownerUserId,
      );
}
