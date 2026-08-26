/// Where the PocketClaw Telegram onboarding service lives.
///
/// This is a build-time setting, supplied with
/// `--dart-define=POCKETCLAW_ONBOARDING_BASE_URL=https://...`, pointing at a
/// PocketClaw-owned deployment of `services/telegram-onboarding`.
///
/// It is deliberately empty by default. No endpoint is guessed and no
/// third-party onboarding service is ever contacted: until an operator deploys
/// PocketClaw's own service and builds with this define, the app reports that
/// automatic setup is unavailable and offers manual token entry instead.
///
/// Only a public base URL belongs here. The manager bot's token is a server
/// secret and never ships in the APK.
abstract final class TelegramOnboardingConfig {
  static const String baseUrl =
      String.fromEnvironment('POCKETCLAW_ONBOARDING_BASE_URL');

  /// Whether automatic managed-bot onboarding can be offered in this build.
  static bool get isConfigured {
    final trimmed = baseUrl.trim();
    if (trimmed.isEmpty) return false;
    final uri = Uri.tryParse(trimmed);
    // Plain HTTP would put the poll token and, once, the bot token on the wire
    // in the clear. Refuse rather than downgrade.
    return uri != null && uri.isScheme('https') && uri.host.isNotEmpty;
  }
}
