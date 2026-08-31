import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'package:pocketclaw/src/core/picoclaw_channel.dart';

/// GitHub authentication for the bundled `gh` and for Git over HTTPS.
///
/// The card shows whether a credential exists and which account it belongs to.
/// It never shows the token, and there is no control that could reveal one:
/// PocketClaw stores it encrypted under an Android Keystore key and hands it to
/// Core only as an environment variable, so nothing above this layer has a copy
/// to display.
class GitHubSettingsCard extends StatefulWidget {
  const GitHubSettingsCard({super.key, this.onCredentialChanged});

  /// Called after connecting or disconnecting. The credential is read when Core
  /// starts, so the change takes effect on the next start.
  final Future<void> Function()? onCredentialChanged;

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
      final connection = await PicoClawChannel.getGitHubStatus();
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
      final connection = await PicoClawChannel.connectGitHub(token);
      if (!mounted) return;
      setState(() => _connection = connection);
      _report(
        'Connected as ${connection.login ?? 'your GitHub account'}. '
        'Restart PocketClaw for gh and git to use it.',
      );
      await widget.onCredentialChanged?.call();
    } on PlatformException catch (error) {
      _report(error.message ?? 'GitHub did not accept this token.',
          isError: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _test() async {
    setState(() {
      _busy = true;
      _message = null;
    });
    try {
      final login = await PicoClawChannel.testGitHubConnection();
      _report(login.isEmpty
          ? 'GitHub authentication is working.'
          : 'Authenticated as $login.');
    } on PlatformException catch (error) {
      _report(error.message ?? 'GitHub authentication is not working.',
          isError: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _disconnect() async {
    setState(() {
      _busy = true;
      _message = null;
    });
    try {
      await PicoClawChannel.disconnectGitHub();
      if (!mounted) return;
      setState(() => _connection = const GitHubConnection(connected: false));
      _report('Disconnected. Restart PocketClaw to complete it.');
      await widget.onCredentialChanged?.call();
    } on PlatformException catch (error) {
      _report(error.message ?? 'Could not remove the credential.',
          isError: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final theme = Theme.of(context);

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
                Icon(Icons.code_rounded, color: scheme.primary),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Text('GitHub'),
                      Text(
                        _loading
                            ? 'Checking…'
                            : _connection.connected
                                ? 'Connected${_connection.login == null ? '' : ' as ${_connection.login}'}'
                                : 'Not connected',
                        key: const Key('github-connection-state'),
                        style: theme.textTheme.bodySmall,
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
            const SizedBox(height: 4),
            Text(
              'Used by the bundled gh and by Git over HTTPS. The token is '
              'encrypted on this device and is never shown again.',
              style: theme.textTheme.bodySmall,
            ),
            if (_message != null) ...[
              const SizedBox(height: 8),
              Text(
                _message!,
                key: const Key('github-card-message'),
                style: theme.textTheme.bodySmall?.copyWith(
                  color: _messageIsError ? scheme.error : scheme.primary,
                ),
              ),
            ],
            const SizedBox(height: 4),
            Align(
              alignment: Alignment.centerRight,
              child: Wrap(
                spacing: 8,
                children: _connection.connected
                    ? [
                        TextButton(
                          key: const Key('github-test-button'),
                          onPressed: _busy ? null : _test,
                          child: const Text('Test connection'),
                        ),
                        TextButton(
                          key: const Key('github-disconnect-button'),
                          onPressed: _busy ? null : _disconnect,
                          child: const Text('Disconnect'),
                        ),
                      ]
                    : [
                        FilledButton(
                          key: const Key('github-connect-button'),
                          onPressed: _busy ? null : _connect,
                          child: const Text('Connect GitHub'),
                        ),
                      ],
              ),
            ),
          ],
        ),
      ),
    );
  }
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
    return AlertDialog(
      title: const Text('Connect GitHub'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Paste a GitHub personal access token with the scopes you need '
            '(repo for private repositories).',
          ),
          const SizedBox(height: 12),
          TextField(
            key: const Key('github-token-field'),
            controller: _controller,
            obscureText: true,
            autocorrect: false,
            enableSuggestions: false,
            decoration: const InputDecoration(
              labelText: 'Personal access token',
              border: OutlineInputBorder(),
            ),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        FilledButton(
          key: const Key('github-token-save'),
          onPressed: () {
            final token = _controller.text.trim();
            _controller.clear();
            Navigator.of(context).pop(token);
          },
          child: const Text('Connect'),
        ),
      ],
    );
  }
}
