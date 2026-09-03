import 'package:flutter/material.dart';

import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_strings.dart';

const String canonicalTelegramConsolePath = '/channels/telegram';

/// A neutral shortcut to Core's authoritative Telegram management page.
///
/// Native Settings deliberately does not infer or cache connection state. It
/// also has no dependency on the pairing launcher: creating or replacing a bot
/// starts only from an explicit action inside Channels -> Telegram.
class TelegramSettingsCard extends StatelessWidget {
  const TelegramSettingsCard({super.key, required this.onManage});

  final Future<void> Function(String path)? onManage;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final l10n = AppLocalizations.of(context)!;
    return Card(
      margin: EdgeInsets.zero,
      child: ListTile(
        key: const Key('telegram-settings-card'),
        leading: Icon(Icons.send_rounded, color: scheme.primary),
        title: const Text(TelegramOnboardingStrings.title),
        subtitle: Text(l10n.manageTelegramConnection),
        trailing: const Icon(Icons.chevron_right_rounded),
        onTap: onManage == null
            ? null
            : () => onManage!(canonicalTelegramConsolePath),
      ),
    );
  }
}
