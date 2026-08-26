import 'package:flutter/material.dart';

import 'package:pocketclaw/src/core/pocketclaw_design.dart';
import 'package:pocketclaw/src/telegram/telegram_connection_status.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_strings.dart';

/// What the native Settings entry opens when Telegram is already configured.
///
/// The defect this replaces: the settings card always read "Connect PocketClaw
/// to Telegram" and opened onboarding, so an already-connected user was
/// offered a brand-new pairing as the default action while the web console
/// correctly showed Connected. Creating a replacement bot is now something the
/// user has to ask for explicitly.
class TelegramConnectedPage extends StatelessWidget {
  const TelegramConnectedPage({
    super.key,
    required this.status,
    required this.onOpenChat,
    required this.onReconnect,
    required this.onAdvanced,
  });

  final TelegramConnectionStatus status;
  final VoidCallback? onOpenChat;
  final VoidCallback onReconnect;
  final VoidCallback onAdvanced;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text(TelegramOnboardingStrings.title)),
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(PocketClawDesign.spaceLarge),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    children: [
                      Icon(Icons.check_circle_rounded, color: scheme.primary),
                      const SizedBox(width: 8),
                      Text(
                        TelegramOnboardingStrings.connected,
                        style: theme.textTheme.titleLarge,
                      ),
                    ],
                  ),
                  const SizedBox(height: 24),
                  Text(
                    TelegramOnboardingStrings.yourBotLabel,
                    style: theme.textTheme.labelMedium,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    status.botUsername == null
                        ? TelegramOnboardingStrings.connectedUnknownBot
                        : '@${status.botUsername}',
                    key: const Key('telegram-connected-bot'),
                    style: theme.textTheme.bodyLarge,
                  ),
                  const SizedBox(height: 16),
                  Text(
                    TelegramOnboardingStrings.ownerLabel,
                    style: theme.textTheme.labelMedium,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    status.ownerRestricted
                        ? TelegramOnboardingStrings.ownerConfigured
                        : TelegramOnboardingStrings.ownerAnyone,
                    style: theme.textTheme.bodyLarge,
                  ),
                  const SizedBox(height: 28),
                  if (onOpenChat != null)
                    FilledButton.icon(
                      onPressed: onOpenChat,
                      icon: const Icon(Icons.send_rounded),
                      label: const Text(TelegramOnboardingStrings.openChat),
                    ),
                  if (onOpenChat != null) const SizedBox(height: 12),
                  OutlinedButton.icon(
                    onPressed: onReconnect,
                    icon: const Icon(Icons.refresh_rounded),
                    label: const Text(TelegramOnboardingStrings.reconnect),
                  ),
                  const SizedBox(height: 12),
                  TextButton(
                    onPressed: onAdvanced,
                    child: const Text(
                      TelegramOnboardingStrings.advancedSettings,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
