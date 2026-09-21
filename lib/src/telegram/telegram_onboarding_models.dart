/// Data types for the PocketClaw Telegram managed-bot onboarding flow.
///
/// These mirror the pairing API in `services/telegram-onboarding`. Nothing
/// here ever holds the manager bot's credentials; the only secret that reaches
/// the app is the user's own child bot token, and only through
/// [TelegramBotCredentials].
library;

import 'telegram_deep_link.dart';

/// The observable state of a pairing session, as reported by the service.
enum PairingState {
  /// The link has been issued and Telegram has reported nothing yet.
  pending,

  /// Telegram reported the new bot; its token is being retrieved.
  created,

  /// The token is held by the service, awaiting its single delivery.
  ready,

  /// The session outlived its window.
  expired,

  /// The session cannot complete.
  failed;

  static PairingState parse(String raw) {
    return PairingState.values.firstWhere(
      (state) => state.name == raw,
      // An unknown state from a newer service is treated as a failure rather
      // than as success. Guessing forward here could mean reporting a bot as
      // connected when it is not.
      orElse: () => PairingState.failed,
    );
  }
}

/// A pairing session, as returned when it is created.
///
/// [pollToken] authenticates this app as the session's owner. It is not the
/// bot token and grants nothing beyond this one pairing.
class TelegramPairing {
  const TelegramPairing({
    required this.pairingId,
    required this.pollToken,
    required this.suggestedUsername,
    required this.suggestedName,
    required this.deepLink,
    required this.qrPayload,
    required this.expiresAt,
    required this.pollInterval,
  });

  final String pairingId;
  final String pollToken;
  final String suggestedUsername;
  final String suggestedName;
  final String deepLink;
  final String qrPayload;
  final DateTime expiresAt;
  final Duration pollInterval;

  bool isExpiredAt(DateTime now) => !now.isBefore(expiresAt);

  factory TelegramPairing.fromJson(Map<String, dynamic> json) {
    return TelegramPairing(
      pairingId: json['pairing_id'] as String,
      pollToken: json['poll_token'] as String,
      suggestedUsername: json['suggested_username'] as String,
      suggestedName: json['suggested_name'] as String,
      deepLink: json['deep_link'] as String,
      qrPayload: json['qr_payload'] as String,
      expiresAt: DateTime.parse(json['expires_at'] as String).toUtc(),
      pollInterval: Duration(
        seconds: (json['poll_interval_seconds'] as num?)?.toInt() ?? 2,
      ),
    );
  }

  /// The subset that is safe to persist so a pairing survives the app being
  /// killed while the user is in Telegram. Deliberately excludes nothing —
  /// every field here is either public or scoped to this one short-lived
  /// session — but it is kept explicit so the choice is visible.
  Map<String, dynamic> toStorageJson() => {
    'pairing_id': pairingId,
    'poll_token': pollToken,
    'suggested_username': suggestedUsername,
    'suggested_name': suggestedName,
    'deep_link': deepLink,
    'qr_payload': qrPayload,
    'expires_at': expiresAt.toIso8601String(),
    'poll_interval_seconds': pollInterval.inSeconds,
  };

  static TelegramPairing? fromStorageJson(Map<String, dynamic> json) {
    try {
      return TelegramPairing.fromJson(json);
    } catch (_) {
      return null;
    }
  }
}

/// The state of a pairing at one point in time.
class TelegramPairingStatus {
  const TelegramPairingStatus({
    required this.state,
    this.reason,
    this.botUsername,
    this.ownerUserId,
  });

  final PairingState state;

  /// A non-secret machine-readable failure reason, when [state] is failed.
  final String? reason;

  /// The created bot's username, known from [PairingState.created] onward.
  final String? botUsername;

  /// The Telegram user who created the bot. It becomes the bot's allow-list.
  final int? ownerUserId;

  factory TelegramPairingStatus.fromJson(Map<String, dynamic> json) {
    final rawOwner = json['owner_user_id'];
    return TelegramPairingStatus(
      state: PairingState.parse(json['state'] as String? ?? 'failed'),
      reason: json['reason'] as String?,
      botUsername: json['bot_username'] as String?,
      ownerUserId: rawOwner is num ? rawOwner.toInt() : null,
    );
  }
}

/// The child bot's credentials, delivered exactly once.
///
/// [token] is the user's own bot token. It goes straight into Core's
/// configuration and is never logged, displayed in full, or sent anywhere
/// else.
class TelegramBotCredentials {
  const TelegramBotCredentials({
    required this.token,
    required this.botUserId,
    required this.botUsername,
    required this.ownerUserId,
  });

  final String token;
  final int botUserId;
  final String botUsername;
  final int ownerUserId;

  /// The bot's chat link, for the Open Chat action.
  ///
  /// PC-DEF-075. Canonicalised rather than interpolated: a leading `@` from the
  /// service would produce `https://t.me/@name`, which Telegram reports as
  /// "Username not found". Null when the username is unusable.
  String? get chatUrl => telegramBotChatUrl(botUsername);

  factory TelegramBotCredentials.fromJson(Map<String, dynamic> json) {
    return TelegramBotCredentials(
      token: json['bot_token'] as String,
      botUserId: (json['bot_user_id'] as num).toInt(),
      botUsername: json['bot_username'] as String,
      ownerUserId: (json['owner_user_id'] as num).toInt(),
    );
  }

  /// Never returns the token. Overriding this is what stops a stray
  /// `print(credentials)` or an error interpolation from leaking it.
  @override
  String toString() =>
      'TelegramBotCredentials(@$botUsername, botUserId: $botUserId, '
      'ownerUserId: $ownerUserId, token: <redacted>)';
}

/// A failure from the onboarding service that the UI can act on.
class TelegramOnboardingException implements Exception {
  const TelegramOnboardingException(this.kind, [this.detail]);

  final TelegramOnboardingErrorKind kind;
  final String? detail;

  @override
  String toString() =>
      'TelegramOnboardingException(${kind.name}${detail == null ? '' : ': $detail'})';
}

/// Why onboarding could not proceed. Each maps to a distinct user-facing
/// message and a distinct retry decision.
enum TelegramOnboardingErrorKind {
  /// No onboarding endpoint has been configured in the app.
  notConfigured,

  /// The service could not be reached.
  network,

  /// The service refused the request rate.
  rateLimited,

  /// The pairing is gone: expired, unknown, or already delivered.
  pairingGone,

  /// The service answered, but not with something usable.
  serviceError,

  /// Telegram is not installed, so the deep link could not be opened.
  telegramUnavailable,

  /// The setup link could not be resolved to a Telegram destination.
  ///
  /// PC-DEF-052. The app opens Telegram, never the service that issued the link,
  /// so a link that does not resolve to Telegram is refused rather than opened.
  /// Distinct from [telegramUnavailable]: there Telegram is missing, here there
  /// was nothing safe to hand it.
  telegramLinkUnavailable,

  /// Core's configuration could not be written or reloaded.
  configurationFailed,

  /// Telegram answered 401 for this bot token. Retrying the same credential
  /// cannot work; the user must finish creation or provide another bot.
  invalidCredentials,

  /// The configuration was saved, but Core never reported the Telegram channel
  /// as running within the wait.
  ///
  /// PC-DEF-056. Distinct from [configurationFailed]: the token and owner are
  /// persisted and sound, so the user is told to retry the start rather than
  /// redo the setup — and is never sent into a bot chat that cannot answer.
  runtimeNotReady,
}
