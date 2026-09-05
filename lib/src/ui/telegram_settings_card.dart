import 'package:flutter/material.dart';

import 'package:pocketclaw/src/core/aperture_theme.dart';
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
    final tokens = context.aperture;
    final l10n = AppLocalizations.of(context)!;
    return Card(
      margin: EdgeInsets.zero,
      child: ListTile(
        key: const Key('telegram-settings-card'),
        leading: Container(
          width: 40,
          height: 40,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: tokens.surface3,
            borderRadius: BorderRadius.circular(ApertureTheme.radiusSm),
          ),
          child: Icon(Icons.send_rounded, color: tokens.accent, size: 20),
        ),
        title: const Text(TelegramOnboardingStrings.title),
        subtitle: Text(l10n.manageTelegramConnection),
        trailing: Icon(Icons.chevron_right_rounded, color: tokens.textFaint),
        onTap: onManage == null
            ? null
            : () => onManage!(canonicalTelegramConsolePath),
      ),
    );
  }
}
