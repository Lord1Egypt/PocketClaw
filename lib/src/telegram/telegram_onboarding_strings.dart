import 'telegram_onboarding_models.dart';

/// User-facing text for the Telegram onboarding screen.
///
/// These live here rather than in `lib/l10n/*.arb` because this repository
/// keeps all twelve locales at full parity, and adding machine-guessed
/// translations for twelve languages would put unverified text in front of
/// users. Collecting the strings in one class keeps the later move to
/// `.arb` mechanical. Tracked in TASKS.md.
abstract final class TelegramOnboardingStrings {
  static const title = 'Telegram';

  static const introHeadline = 'Connect PocketClaw to Telegram';
  static const introBody =
      'Create your personal PocketClaw bot in a few seconds. '
      'You confirm the new bot in Telegram; PocketClaw sets up the rest.';
  static const connect = 'Connect Telegram';

  static const creatingPairing = 'Preparing your setup link…';

  static const openTelegram = 'Open Telegram';
  static const scanInstead =
      'On another device? Scan this with your phone’s camera.';
  static const waiting = 'Waiting for Telegram…';
  static const waitingBody =
      'Confirm the new bot in Telegram, then come back here.';
  static const suggestedBotLabel = 'Suggested bot';

  static const botCreated = 'Bot created';
  static const configuring = 'Configuring PocketClaw…';

  static const connected = 'Connected';
  static const connectedBody = 'Your PocketClaw bot is ready.';
  static const yourBotLabel = 'Your bot';
  static const openChat = 'Open Chat';
  static const done = 'Done';

  static const expiredHeadline = 'Setup link expired';
  static const expiredBody =
      'The link is only valid for a few minutes. Start again to get a new one.';

  static const tryAgain = 'Try again';
  static const manualSetup = 'Set up manually';
  static const manualPrompt = 'Having trouble connecting automatically?';

  static const manualTitle = 'Manual Telegram setup';
  static const manualBody =
      'Create a bot with @BotFather in Telegram, then paste its token here.';
  static const manualTokenLabel = 'Bot token';
  static const manualAllowedLabel = 'Allowed Telegram user IDs (optional)';
  static const manualAllowedHelp = 'Comma-separated. Leave empty to allow anyone.';
  static const manualSave = 'Save and connect';
  static const manualTokenRequired = 'Enter the bot token from @BotFather.';
  static const manualTokenMalformed =
      'That does not look like a bot token. It looks like 123456789:AA…';

  static const cancel = 'Cancel';

  /// A user-facing explanation for each failure, and nothing more. Service
  /// detail is deliberately not surfaced: it can quote a request.
  static String errorMessage(TelegramOnboardingErrorKind? kind) {
    switch (kind) {
      case TelegramOnboardingErrorKind.notConfigured:
        return 'Automatic setup is not available in this build. '
            'You can still set up Telegram manually.';
      case TelegramOnboardingErrorKind.network:
        return 'PocketClaw could not reach the setup service. '
            'Check your connection and try again.';
      case TelegramOnboardingErrorKind.rateLimited:
        return 'Too many setup attempts. Wait a moment and try again.';
      case TelegramOnboardingErrorKind.pairingGone:
        return 'This setup link is no longer valid. Start again to get a new one.';
      case TelegramOnboardingErrorKind.telegramUnavailable:
        return 'PocketClaw could not open Telegram. '
            'Install Telegram, or scan the QR code from another device.';
      case TelegramOnboardingErrorKind.configurationFailed:
        return 'The bot was created, but PocketClaw could not finish '
            'configuring it. Try again, or set it up manually.';
      case TelegramOnboardingErrorKind.serviceError:
      case null:
        return 'Something went wrong during setup. Try again, '
            'or set it up manually.';
    }
  }
}
