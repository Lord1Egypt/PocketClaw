import 'package:flutter/material.dart';

import 'package:pocketclaw/src/telegram/telegram_connection_status.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_strings.dart';
import 'package:pocketclaw/src/ui/telegram_onboarding_launcher.dart';

/// The Telegram entry in native Settings.
///
/// It renders from the persisted `channel_list.telegram` entry — the same
/// state the embedded console reads — instead of a hardcoded "not connected"
/// label. Before this, an already-connected user saw "Connect PocketClaw to
/// Telegram" here while the console correctly showed Connected, and tapping it
/// offered a brand-new pairing.
class TelegramSettingsCard extends StatefulWidget {
  const TelegramSettingsCard({super.key, this.readStatus, this.onOpen});

  /// Overridable so a test can drive real configuration shapes through the
  /// same widget the app builds.
  final Future<TelegramConnectionStatus> Function()? readStatus;

  /// Overridable so a test can observe what tapping actually does.
  final Future<void> Function(BuildContext context)? onOpen;

  @override
  State<TelegramSettingsCard> createState() => _TelegramSettingsCardState();
}

class _TelegramSettingsCardState extends State<TelegramSettingsCard>
    with WidgetsBindingObserver {
  TelegramConnectionStatus? _status;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _refresh();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    // Telegram can also be configured from the embedded console or reset
    // outside this screen; re-read rather than trusting what was loaded once.
    if (state == AppLifecycleState.resumed) _refresh();
  }

  Future<void> _refresh() async {
    final read = widget.readStatus ?? TelegramOnboardingLauncher.readStatus;
    final status = await read();
    if (!mounted) return;
    setState(() => _status = status);
  }

  Future<void> _open() async {
    final open = widget.onOpen ?? TelegramOnboardingLauncher.open;
    await open(context);
    // The flow may have connected, replaced, or reset the bot.
    await _refresh();
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final status = _status;
    final connected = status?.configured ?? false;

    final String subtitle;
    if (status == null) {
      subtitle = TelegramOnboardingStrings.introHeadline;
    } else if (!connected) {
      subtitle = TelegramOnboardingStrings.introHeadline;
    } else if (status.botUsername != null) {
      subtitle = '${TelegramOnboardingStrings.connectedSubtitle} · '
          '@${status.botUsername}';
    } else {
      subtitle = TelegramOnboardingStrings.connectedSubtitle;
    }

    return Card(
      margin: EdgeInsets.zero,
      child: ListTile(
        key: const Key('telegram-settings-card'),
        leading: Icon(
          connected ? Icons.check_circle_rounded : Icons.send_rounded,
          color: scheme.primary,
        ),
        title: const Text(TelegramOnboardingStrings.title),
        subtitle: Text(subtitle),
        trailing: const Icon(Icons.chevron_right_rounded),
        onTap: _open,
      ),
    );
  }
}
