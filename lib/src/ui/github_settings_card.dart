import 'package:flutter/material.dart';

import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:flutter/services.dart';

import 'package:pocketclaw/src/core/pocketclaw_channel.dart';
import 'package:pocketclaw/src/core/service_manager.dart';

/// GitHub authentication for the bundled `gh` and for Git over HTTPS.
///
/// The card shows whether a credential exists and which account it belongs to.
/// It never shows the token, and there is no control that could reveal one:
/// PocketClaw stores it encrypted under an Android Keystore key and hands it to
/// Core only as an environment variable, so nothing above this layer has a copy
/// to display.
class GitHubSettingsCard extends StatefulWidget {
  const GitHubSettingsCard({super.key, this.onCredentialChanged});

  /// Applies the change to a running Core. The credential is read when Core
  /// launches, so a restart is what makes it live; the caller supplies the
  /// app's existing safe restart rather than this card inventing one.
  final Future<CredentialApplyOutcome> Function()? onCredentialChanged;

  @override
  State<GitHubSettingsCard> createState() => _GitHubSettingsCardState();
}

class _GitHubSettingsCardState extends State<GitHubSettingsCard> {
  GitHubConnection _connection = const GitHubConnection(connected: false);
  bool _loading = true;
  bool _busy = false;
  String? _message;
  bool _messageIsError = false;

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    try {
      final connection = await PocketClawChannel.getGitHubStatus();
      if (!mounted) return;
      setState(() {
        _connection = connection;
        _loading = false;
      });
    } on PlatformException {
      if (!mounted) return;
      setState(() => _loading = false);
    }
  }

  void _report(String message, {bool isError = false}) {
    if (!mounted) return;
    setState(() {
      _message = message;
      _messageIsError = isError;
    });
  }

  Future<void> _connect() async {
    // Captured before the first await: reading it afterwards would be a
    // BuildContext use across an async gap.
    final l10n = AppLocalizations.of(context)!;
    final token = await showDialog<String>(
      context: context,
      builder: (context) => const _GitHubTokenDialog(),
    );
    if (token == null || token.isEmpty) return;

    setState(() {
      _busy = true;
      _message = null;
    });
    try {
      final connection = await PocketClawChannel.connectGitHub(token);
      if (!mounted) return;
      setState(() => _connection = connection);
      final who = connection.login ?? l10n.githubYourAccount;
      _report(l10n.githubConnectedReport(who, await _applyOutcome(l10n)));
    } on PlatformException catch (error) {
      _report(error.message ?? l10n.githubTokenRejected, isError: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _test() async {
    // Captured before the first await: reading it afterwards would be a
    // BuildContext use across an async gap.
    final l10n = AppLocalizations.of(context)!;
    setState(() {
      _busy = true;
      _message = null;
    });
    try {
      final login = await PocketClawChannel.testGitHubConnection();
      _report(
        login.isEmpty
            ? l10n.githubAuthWorking
            : l10n.githubAuthenticatedAs(login),
      );
    } on PlatformException catch (error) {
      _report(error.message ?? l10n.githubAuthNotWorking, isError: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _disconnect() async {
    // Captured before the first await: reading it afterwards would be a
    // BuildContext use across an async gap.
    final l10n = AppLocalizations.of(context)!;
    setState(() {
      _busy = true;
      _message = null;
    });
    try {
      await PocketClawChannel.disconnectGitHub();
      if (!mounted) return;
      setState(() => _connection = const GitHubConnection(connected: false));
      _report(l10n.githubDisconnectedReport(await _applyOutcome(l10n)));
    } on PlatformException catch (error) {
      _report(
        error.message ?? l10n.githubCredentialRemoveFailed,
        isError: true,
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  /// Applies the change through the app's normal restart and says what
  /// happened, because "saved" and "in effect" are different states and the
  /// user is about to run a gh command on the strength of this message.
  Future<String> _applyOutcome(AppLocalizations l10n) async {
    final apply = widget.onCredentialChanged;
    if (apply == null) {
      return l10n.credentialAppliesNextStart;
    }
    switch (await apply()) {
      case CredentialApplyOutcome.applied:
        return l10n.credentialAppliedNow;
      case CredentialApplyOutcome.notRunning:
        return l10n.credentialAppliesNextStart;
      case CredentialApplyOutcome.deferred:
        return l10n.credentialAppliesDeferred;
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final tokens = context.aperture;
    final l10n = AppLocalizations.of(context)!;

    return Card(
      key: const Key('github-settings-card'),
      margin: EdgeInsets.zero,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  width: 40,
                  height: 40,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: tokens.surface3,
                    borderRadius: BorderRadius.circular(ApertureTheme.radiusSm),
                  ),
                  child: Icon(
                    Icons.code_rounded,
                    color: tokens.accent,
                    size: 20,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'GitHub',
                        style: theme.textTheme.titleSmall?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      // Connection is state, so it carries a dot and a word.
                      // The dot never speaks alone.
                      Row(
                        children: [
                          Container(
                            width: 6,
                            height: 6,
                            margin: const EdgeInsetsDirectional.only(end: 6),
                            decoration: BoxDecoration(
                              shape: BoxShape.circle,
                              color: _loading
                                  ? tokens.textFaint
                                  : _connection.connected
                                  ? tokens.success
                                  : tokens.textFaint,
                            ),
                          ),
                          Expanded(child: _connectionText(l10n, theme)),
                        ],
                      ),
                    ],
                  ),
                ),
                if (_busy)
                  const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  ),
              ],
            ),
            const SizedBox(height: 8),
            Text(l10n.githubDescription, style: theme.textTheme.bodySmall),
            if (_message != null) ...[
              const SizedBox(height: 8),
              Text(
                _message!,
                key: const Key('github-card-message'),
                style: theme.textTheme.bodySmall?.copyWith(
                  color: _messageIsError ? tokens.danger : tokens.accent,
                ),
              ),
            ],
            const SizedBox(height: 4),
            Align(
              alignment: AlignmentDirectional.centerEnd,
              child: Wrap(
                spacing: 8,
                children: _connection.connected
                    ? [
                        TextButton(
                          key: const Key('github-test-button'),
                          onPressed: _busy ? null : _test,
                          child: Text(l10n.githubTestConnection),
                        ),
                        TextButton(
                          key: const Key('github-disconnect-button'),
                          onPressed: _busy ? null : _disconnect,
                          child: Text(l10n.githubDisconnect),
                        ),
                      ]
                    : [
                        FilledButton(
                          key: const Key('github-connect-button'),
                          onPressed: _busy ? null : _connect,
                          child: Text(l10n.githubConnectAction),
                        ),
                      ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// The connection sentence. The widget key is part of the test contract and
  /// does not move.
  Widget _connectionText(AppLocalizations l10n, ThemeData theme) => Text(
    _loading
        ? l10n.githubChecking
        : _connection.connected
        ? (_connection.login == null
              ? l10n.githubConnected
              : l10n.githubConnectedAs(_connection.login!))
        : l10n.githubNotConnected,
    key: const Key('github-connection-state'),
    style: theme.textTheme.bodySmall,
  );
}

/// Collects a personal access token and returns it once.
///
/// The field is obscured and the controller is disposed with the dialog, so the
/// value exists in UI state only for as long as it takes to hand it to the
/// platform channel.
class _GitHubTokenDialog extends StatefulWidget {
  const _GitHubTokenDialog();

  @override
  State<_GitHubTokenDialog> createState() => _GitHubTokenDialogState();
}

class _GitHubTokenDialogState extends State<_GitHubTokenDialog> {
  final TextEditingController _controller = TextEditingController();

  @override
  void dispose() {
    _controller.clear();
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return AlertDialog(
      title: Text(l10n.githubConnectAction),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(l10n.githubTokenHint),
          const SizedBox(height: 12),
          TextField(
            key: const Key('github-token-field'),
            controller: _controller,
            obscureText: true,
            autocorrect: false,
            enableSuggestions: false,
            decoration: InputDecoration(
              labelText: l10n.githubTokenLabel,
              border: const OutlineInputBorder(),
            ),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(l10n.cancel),
        ),
        FilledButton(
          key: const Key('github-token-save'),
          onPressed: () {
            final token = _controller.text.trim();
            _controller.clear();
            Navigator.of(context).pop(token);
          },
          child: Text(l10n.githubConnect),
        ),
      ],
    );
  }
}
