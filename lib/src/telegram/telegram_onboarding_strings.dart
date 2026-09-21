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
  static const startingRuntime = 'Starting Telegram…';

  static const connected = 'Connected';
  static const connectedBody = 'Your PocketClaw bot is ready.';
  static const yourBotLabel = 'Your bot';
  static const openChat = 'Open Chat';
  static const done = 'Done';

  // Connected-state surface, shown when Telegram is already configured.
  static const connectedSubtitle = 'Connected';
  static const connectedUnknownBot = 'Bot configured on this device';
  static const ownerLabel = 'Owner';
  static const ownerConfigured = 'Configured';
  static const ownerAnyone = 'Anyone can message this bot';
  static const reconnect = 'Reconnect / Create New Bot';
  static const advancedSettings = 'Advanced / Manual Settings';
  static const reconnectConfirmTitle = 'Create a new bot?';
  static const reconnectConfirmBody =
      'Your current bot keeps working until a new one is created. '
      'Only when the new bot is ready does PocketClaw switch over. '
      'If you cancel or the link expires, nothing changes.';
  static const reconnectConfirmCancel = 'Keep current bot';
  static const reconnectConfirmProceed = 'Create new bot';

  static const expiredHeadline = 'Setup link expired';
  static const expiredBody =
      'The link is only valid for a few minutes. Start again to get a new one.';

  static const tryAgain = 'Try again';
  static const connectionFailed = 'Telegram connection failed';
  static const createOrReplaceBot = 'Create or replace bot';
  static const manualSetup = 'Set up manually';
  static const manualPrompt = 'Having trouble connecting automatically?';

  static const manualTitle = 'Manual Telegram setup';
  static const manualBody =
      'Create a bot with @BotFather in Telegram, then paste its token here.';
  static const manualTokenLabel = 'Bot token';
  static const manualAllowedLabel = 'Owner Telegram numeric user ID';
  static const manualAllowedHelp =
      'Required. Find your numeric ID from a trusted Telegram ID bot.';
  static const manualSave = 'Save and connect';
  static const manualTokenRequired = 'Enter the bot token from @BotFather.';
  static const manualOwnerRequired = 'Enter your numeric Telegram user ID.';
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
        return 'Telegram is limiting bot creation. Try again later, '
            'or set up an existing valid bot manually.';
      case TelegramOnboardingErrorKind.pairingGone:
        return 'This setup link is no longer valid. Start again to get a new one.';
      case TelegramOnboardingErrorKind.telegramUnavailable:
        return 'PocketClaw could not open Telegram. '
            'Install Telegram, or scan the QR code from another device.';
      case TelegramOnboardingErrorKind.telegramLinkUnavailable:
        return 'PocketClaw could not prepare your Telegram setup link. '
            'Try again, or set it up manually.';
      case TelegramOnboardingErrorKind.configurationFailed:
        return 'The bot was created, but PocketClaw could not finish '
            'configuring it. Try again, or set it up manually.';
      case TelegramOnboardingErrorKind.invalidCredentials:
        return 'Telegram rejected this bot. Bot creation may be incomplete, '
            'or its token is no longer valid. Try again later, create or replace '
            'the bot, or set up an existing valid bot manually.';
      case TelegramOnboardingErrorKind.runtimeNotReady:
        return 'Your bot is saved, but Telegram has not started yet. '
            'Open PocketClaw and try again in a moment.';
      case TelegramOnboardingErrorKind.serviceError:
      case null:
        return 'Something went wrong during setup. Try again, '
            'or set it up manually.';
    }
  }
}
