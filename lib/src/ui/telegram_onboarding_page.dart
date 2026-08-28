import 'package:flutter/material.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../core/pocketclaw_design.dart';
import '../telegram/telegram_config_writer.dart';
import '../telegram/telegram_onboarding_controller.dart';
import '../telegram/telegram_onboarding_models.dart';
import '../telegram/telegram_onboarding_strings.dart';

/// PocketClaw's Telegram setup screen.
///
/// The default path never shows the user a token. They tap Open Telegram (or
/// scan the QR from another device), confirm the bot Telegram offers them, and
/// come back to a configured channel. Manual token entry stays available
/// behind "Set up manually".
class TelegramOnboardingPage extends StatefulWidget {
  const TelegramOnboardingPage({
    super.key,
    required this.controller,
    required this.configWriter,
    this.onManualConfigurationSaved,
  });

  final TelegramOnboardingController controller;

  /// Used by the manual fallback, which writes the same configuration the
  /// automatic flow does.
  final TelegramConfigWriter configWriter;
  final Future<void> Function()? onManualConfigurationSaved;

  @override
  State<TelegramOnboardingPage> createState() => _TelegramOnboardingPageState();
}

class _TelegramOnboardingPageState extends State<TelegramOnboardingPage>
    with WidgetsBindingObserver {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    widget.controller.addListener(_onControllerChanged);
    // A pairing may have outlived a process death while the user was in
    // Telegram; pick it back up rather than making them start over.
    widget.controller.restore();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    widget.controller.removeListener(_onControllerChanged);
    super.dispose();
  }

  void _onControllerChanged() {
    if (mounted) setState(() {});
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    // Telegram taking focus must not end the pairing. Polling stops while the
    // app is backgrounded and resumes with an immediate check on return.
    switch (state) {
      case AppLifecycleState.paused:
      case AppLifecycleState.hidden:
      case AppLifecycleState.detached:
        widget.controller.pausePolling();
      case AppLifecycleState.resumed:
        widget.controller.resumePolling();
      case AppLifecycleState.inactive:
        break;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text(TelegramOnboardingStrings.title)),
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(PocketClawDesign.spaceLarge),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: _buildStage(context),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildStage(BuildContext context) {
    switch (widget.controller.stage) {
      case TelegramOnboardingStage.idle:
        return _buildIntro(context);
      case TelegramOnboardingStage.creatingPairing:
        return _buildBusy(context, TelegramOnboardingStrings.creatingPairing);
      case TelegramOnboardingStage.awaitingConfirmation:
        return _buildAwaitingConfirmation(context);
      case TelegramOnboardingStage.botCreated:
        return _buildBusy(
          context,
          TelegramOnboardingStrings.botCreated,
          detail: TelegramOnboardingStrings.configuring,
        );
      case TelegramOnboardingStage.configuring:
        return _buildBusy(context, TelegramOnboardingStrings.configuring);
      case TelegramOnboardingStage.connected:
        return _buildConnected(context);
      case TelegramOnboardingStage.expired:
        return _buildExpired(context);
      case TelegramOnboardingStage.failed:
        return _buildFailed(context);
    }
  }

  Widget _buildIntro(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Icon(Icons.send_rounded, size: 56, color: theme.colorScheme.primary),
        const SizedBox(height: PocketClawDesign.spaceLarge),
        Text(
          TelegramOnboardingStrings.introHeadline,
          style: theme.textTheme.headlineSmall,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: PocketClawDesign.spaceMedium),
        Text(
          TelegramOnboardingStrings.introBody,
          style: theme.textTheme.bodyMedium,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: PocketClawDesign.spaceLarge),
        FilledButton.icon(
          onPressed: widget.controller.start,
          icon: const Icon(Icons.link_rounded),
          label: const Text(TelegramOnboardingStrings.connect),
        ),
        const SizedBox(height: PocketClawDesign.spaceMedium),
        _buildManualFallback(context),
      ],
    );
  }

  Widget _buildBusy(BuildContext context, String message, {String? detail}) {
    final theme = Theme.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        const CircularProgressIndicator(),
        const SizedBox(height: PocketClawDesign.spaceLarge),
        Text(
          message,
          style: theme.textTheme.titleMedium,
          textAlign: TextAlign.center,
        ),
        if (detail != null) ...[
          const SizedBox(height: PocketClawDesign.spaceSmall),
          Text(
            detail,
            style: theme.textTheme.bodySmall,
            textAlign: TextAlign.center,
          ),
        ],
      ],
    );
  }

  Widget _buildAwaitingConfirmation(BuildContext context) {
    final theme = Theme.of(context);
    final pairing = widget.controller.pairing;
    if (pairing == null) return _buildIntro(context);

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        FilledButton.icon(
          onPressed: widget.controller.openTelegram,
          icon: const Icon(Icons.open_in_new_rounded),
          label: const Text(TelegramOnboardingStrings.openTelegram),
        ),
        const SizedBox(height: PocketClawDesign.spaceLarge),
        Text(
          TelegramOnboardingStrings.scanInstead,
          style: theme.textTheme.bodySmall,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: PocketClawDesign.spaceMedium),
        Center(child: TelegramPairingQr(payload: pairing.qrPayload)),
        const SizedBox(height: PocketClawDesign.spaceLarge),
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const SizedBox(
              width: 16,
              height: 16,
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
            const SizedBox(width: PocketClawDesign.spaceSmall),
            Text(
              TelegramOnboardingStrings.waiting,
              style: theme.textTheme.titleSmall,
            ),
          ],
        ),
        const SizedBox(height: PocketClawDesign.spaceSmall),
        Text(
          TelegramOnboardingStrings.waitingBody,
          style: theme.textTheme.bodySmall,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: PocketClawDesign.spaceMedium),
        _buildLabelledValue(
          context,
          TelegramOnboardingStrings.suggestedBotLabel,
          '@${pairing.suggestedUsername}',
        ),
        _buildExpiryNotice(context),
        const SizedBox(height: PocketClawDesign.spaceMedium),
        TextButton(
          onPressed: widget.controller.retry,
          child: const Text(TelegramOnboardingStrings.tryAgain),
        ),
        _buildManualFallback(context),
      ],
    );
  }

  Widget _buildExpiryNotice(BuildContext context) {
    final remaining = widget.controller.timeRemaining;
    if (remaining == null) return const SizedBox.shrink();
    final minutes = remaining.inMinutes;
    final seconds = remaining.inSeconds % 60;
    return Padding(
      padding: const EdgeInsets.only(top: PocketClawDesign.spaceSmall),
      child: Text(
        'Expires in ${minutes.toString().padLeft(2, '0')}:'
        '${seconds.toString().padLeft(2, '0')}',
        style: Theme.of(context).textTheme.bodySmall,
        textAlign: TextAlign.center,
      ),
    );
  }

  Widget _buildConnected(BuildContext context) {
    final theme = Theme.of(context);
    final username = widget.controller.connectedBotUsername;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Icon(
          Icons.check_circle_rounded,
          size: 56,
          color: theme.colorScheme.primary,
        ),
        const SizedBox(height: PocketClawDesign.spaceLarge),
        Text(
          TelegramOnboardingStrings.connected,
          style: theme.textTheme.headlineSmall,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: PocketClawDesign.spaceSmall),
        Text(
          TelegramOnboardingStrings.connectedBody,
          style: theme.textTheme.bodyMedium,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: PocketClawDesign.spaceLarge),
        if (username != null)
          _buildLabelledValue(
            context,
            TelegramOnboardingStrings.yourBotLabel,
            '@$username',
          ),
        const SizedBox(height: PocketClawDesign.spaceLarge),
        FilledButton.icon(
          onPressed: widget.controller.openBotChat,
          icon: const Icon(Icons.chat_bubble_outline_rounded),
          label: const Text(TelegramOnboardingStrings.openChat),
        ),
        const SizedBox(height: PocketClawDesign.spaceSmall),
        TextButton(
          onPressed: () => Navigator.of(context).maybePop(),
          child: const Text(TelegramOnboardingStrings.done),
        ),
      ],
    );
  }

  Widget _buildExpired(BuildContext context) {
    return _buildProblem(
      context,
      icon: Icons.timer_off_rounded,
      headline: TelegramOnboardingStrings.expiredHeadline,
      body: TelegramOnboardingStrings.expiredBody,
    );
  }

  Widget _buildFailed(BuildContext context) {
    return _buildProblem(
      context,
      icon: Icons.error_outline_rounded,
      headline: TelegramOnboardingStrings.tryAgain,
      body: TelegramOnboardingStrings.errorMessage(widget.controller.errorKind),
    );
  }

  Widget _buildProblem(
    BuildContext context, {
    required IconData icon,
    required String headline,
    required String body,
  }) {
    final theme = Theme.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Icon(icon, size: 48, color: theme.colorScheme.error),
        const SizedBox(height: PocketClawDesign.spaceMedium),
        Text(
          headline,
          style: theme.textTheme.titleLarge,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: PocketClawDesign.spaceSmall),
        Text(
          body,
          style: theme.textTheme.bodyMedium,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: PocketClawDesign.spaceLarge),
        FilledButton(
          onPressed: widget.controller.retry,
          child: const Text(TelegramOnboardingStrings.tryAgain),
        ),
        const SizedBox(height: PocketClawDesign.spaceSmall),
        _buildManualFallback(context),
      ],
    );
  }

  Widget _buildLabelledValue(BuildContext context, String label, String value) {
    final theme = Theme.of(context);
    return Column(
      children: [
        Text(label, style: theme.textTheme.labelMedium),
        const SizedBox(height: 2),
        SelectableText(
          value,
          style: theme.textTheme.titleMedium,
          textAlign: TextAlign.center,
        ),
      ],
    );
  }

  Widget _buildManualFallback(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      children: [
        const SizedBox(height: PocketClawDesign.spaceMedium),
        Text(
          TelegramOnboardingStrings.manualPrompt,
          style: theme.textTheme.bodySmall,
          textAlign: TextAlign.center,
        ),
        TextButton(
          onPressed: () => _openManualSetup(context),
          child: const Text(TelegramOnboardingStrings.manualSetup),
        ),
      ],
    );
  }

  Future<void> _openManualSetup(BuildContext context) async {
    final saved = await showDialog<bool>(
      context: context,
      builder: (context) =>
          TelegramManualSetupDialog(configWriter: widget.configWriter),
    );
    if (saved == true && mounted) {
      await widget.onManualConfigurationSaved?.call();
      if (!mounted) return;
      await widget.controller.reset();
    }
  }
}

/// The pairing QR.
///
/// A named widget rather than an inline `QrImageView` so that what it encodes
/// is assertable: `QrImageView` keeps its payload private, and "the QR carries
/// no secret" is a claim that has to be testable rather than reviewed once.
class TelegramPairingQr extends StatelessWidget {
  const TelegramPairingQr({super.key, required this.payload});

  /// The Telegram managed-bot creation link. Public by construction: it holds
  /// no poll token and no bot token.
  final String payload;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(PocketClawDesign.spaceMedium),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(PocketClawDesign.radiusMedium),
      ),
      child: QrImageView(
        data: payload,
        version: QrVersions.auto,
        size: 200,
        backgroundColor: Colors.white,
      ),
    );
  }
}

/// The advanced path: paste a token from @BotFather.
///
/// It writes the same `channel_list.telegram` entry the automatic flow does,
/// so both paths converge on one Telegram implementation.
class TelegramManualSetupDialog extends StatefulWidget {
  const TelegramManualSetupDialog({super.key, required this.configWriter});

  final TelegramConfigWriter configWriter;

  @override
  State<TelegramManualSetupDialog> createState() =>
      _TelegramManualSetupDialogState();
}

class _TelegramManualSetupDialogState extends State<TelegramManualSetupDialog> {
  final _tokenController = TextEditingController();
  final _allowedController = TextEditingController();
  String? _error;
  bool _saving = false;

  @override
  void dispose() {
    _tokenController.dispose();
    _allowedController.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    final token = _tokenController.text.trim();
    if (token.isEmpty) {
      setState(() => _error = TelegramOnboardingStrings.manualTokenRequired);
      return;
    }
    if (!_looksLikeBotToken(token)) {
      setState(() => _error = TelegramOnboardingStrings.manualTokenMalformed);
      return;
    }

    final owner = _firstAllowedUserId();
    if (owner <= 0) {
      setState(() => _error = TelegramOnboardingStrings.manualOwnerRequired);
      return;
    }

    setState(() {
      _saving = true;
      _error = null;
    });

    try {
      await widget.configWriter.apply(
        TelegramBotCredentials(
          token: token,
          botUserId: _botUserIdFromToken(token),
          botUsername: 'manual',
          ownerUserId: owner,
        ),
      );
      if (mounted) Navigator.of(context).pop(true);
    } on TelegramOnboardingException catch (error) {
      if (!mounted) return;
      setState(() {
        _saving = false;
        _error = TelegramOnboardingStrings.errorMessage(error.kind);
      });
    }
  }

  /// Telegram bot tokens are `<bot id>:<secret>`. Checking the shape catches
  /// the common paste mistakes without ever sending the value anywhere.
  static bool _looksLikeBotToken(String token) {
    final parts = token.split(':');
    if (parts.length != 2) return false;
    if (int.tryParse(parts[0]) == null) return false;
    return parts[1].length >= 20;
  }

  static int _botUserIdFromToken(String token) =>
      int.tryParse(token.split(':').first) ?? 0;

  int _firstAllowedUserId() {
    for (final part in _allowedController.text.split(',')) {
      final parsed = int.tryParse(part.trim());
      if (parsed != null && parsed > 0) return parsed;
    }
    return 0;
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text(TelegramOnboardingStrings.manualTitle),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text(TelegramOnboardingStrings.manualBody),
            const SizedBox(height: PocketClawDesign.spaceMedium),
            TextField(
              controller: _tokenController,
              autocorrect: false,
              enableSuggestions: false,
              obscureText: true,
              decoration: const InputDecoration(
                labelText: TelegramOnboardingStrings.manualTokenLabel,
              ),
            ),
            const SizedBox(height: PocketClawDesign.spaceMedium),
            TextField(
              controller: _allowedController,
              keyboardType: TextInputType.text,
              decoration: const InputDecoration(
                labelText: TelegramOnboardingStrings.manualAllowedLabel,
                helperText: TelegramOnboardingStrings.manualAllowedHelp,
                helperMaxLines: 2,
              ),
            ),
            if (_error != null) ...[
              const SizedBox(height: PocketClawDesign.spaceMedium),
              Text(
                _error!,
                style: TextStyle(color: Theme.of(context).colorScheme.error),
              ),
            ],
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _saving ? null : () => Navigator.of(context).pop(false),
          child: const Text(TelegramOnboardingStrings.cancel),
        ),
        FilledButton(
          onPressed: _saving ? null : _save,
          child: const Text(TelegramOnboardingStrings.manualSave),
        ),
      ],
    );
  }
}
