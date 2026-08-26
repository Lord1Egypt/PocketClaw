import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/gestures.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:webview_flutter/webview_flutter.dart';

import 'package:pocketclaw/src/telegram/telegram_onboarding_config.dart';
import 'package:pocketclaw/src/ui/telegram_onboarding_launcher.dart';
import 'package:pocketclaw/src/ui/webview/pocketclaw_host_bridge.dart';
import 'webview_nav_bar.dart';

class WebViewAndroid extends StatefulWidget {
  final String url;

  const WebViewAndroid({super.key, required this.url});

  @override
  State<WebViewAndroid> createState() => _WebViewAndroidState();
}

class _WebViewAndroidState extends State<WebViewAndroid> {
  WebViewController? _controller;
  bool _isLoading = true;
  String? _telegramBotUsername;
  String? _loadedUrl;

  /// Whether the currently loaded page is the local Core console.
  ///
  /// The console is the only origin the host contract is meant for. The
  /// WebView will follow an outbound link if the user taps one, and handing a
  /// third-party page `openTelegramOnboarding` and `openExternal` would let it
  /// drive the app, so injection is scoped to the console's own origin.
  bool get _isConsoleOrigin =>
      PocketClawHostBridge.isSameOrigin(_loadedUrl, widget.url);

  /// Publishes the host contract into the page.
  ///
  /// Runs on every page load because the console is a single-page app served
  /// fresh on each navigation into the tab, and again after onboarding so the
  /// Telegram surface re-renders as connected.
  Future<void> _injectHost() async {
    final controller = _controller;
    if (controller == null || !_isConsoleOrigin) return;
    try {
      await controller.runJavaScript(
        PocketClawHostBridge.bootstrapScript(
          onboardingConfigured: TelegramOnboardingConfig.isConfigured,
          telegramBotUsername: _telegramBotUsername,
        ),
      );
    } catch (_) {
      // The page can go away mid-injection; the next load re-publishes.
    }
  }

  Future<void> _handleHostMessage(String raw) async {
    final request = PocketClawHostBridge.parseMessage(raw);
    if (request == null) return;

    switch (request.kind) {
      case HostRequestKind.openTelegramOnboarding:
        if (!mounted || !_isConsoleOrigin) return;
        final username = await TelegramOnboardingLauncher.open(context);
        if (username != null && username.isNotEmpty) {
          _telegramBotUsername = username;
        }
        if (!mounted) return;
        await _injectHost();
        await _controller?.runJavaScript(
          PocketClawHostBridge.telegramUpdatedScript,
        );
      case HostRequestKind.openExternal:
        final url = request.url;
        if (url == null || !_isConsoleOrigin) return;
        await launchUrl(Uri.parse(url), mode: LaunchMode.externalApplication);
    }
  }

  @override
  void initState() {
    super.initState();
    unawaited(
      TelegramOnboardingLauncher.readConnectedBotUsername().then((username) {
        if (!mounted || username == null) return;
        _telegramBotUsername = username;
        unawaited(_injectHost());
      }),
    );
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setBackgroundColor(Colors.white)
      ..addJavaScriptChannel(
        PocketClawHostBridge.channelName,
        onMessageReceived: (message) =>
            unawaited(_handleHostMessage(message.message)),
      )
      ..setNavigationDelegate(
        NavigationDelegate(
          onPageStarted: (_) {
            if (mounted) setState(() => _isLoading = true);
          },
          onProgress: (progress) {
            if (progress < 100 && mounted) setState(() => _isLoading = true);
          },
          onPageFinished: (url) {
            _loadedUrl = url;
            unawaited(_injectHost());
            if (mounted) setState(() => _isLoading = false);
          },
          onWebResourceError: (_) {
            if (mounted) setState(() => _isLoading = false);
          },
          onNavigationRequest: (_) => NavigationDecision.navigate,
          onUrlChange: (change) {
            _loadedUrl = change.url ?? _loadedUrl;
            if (mounted && _isLoading) setState(() => _isLoading = false);
          },
        ),
      );
    _controller!.loadRequest(Uri.parse(widget.url));
  }

  @override
  Widget build(BuildContext context) {
    return Stack(
      fit: StackFit.expand,
      children: [
        Container(
          color: Colors.white,
          width: double.infinity,
          height: double.infinity,
          child: Listener(
            onPointerDown: (event) {
              if (event.kind == PointerDeviceKind.mouse &&
                  event.buttons == kSecondaryMouseButton) {
                // Block right-click context menu
              }
            },
            child: _controller == null
                ? Container(
                    color: Colors.grey[200],
                    width: double.infinity,
                    height: double.infinity,
                    child: const Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(Icons.web, size: 48, color: Colors.grey),
                          SizedBox(height: 16),
                          Text('Initializing WebView...'),
                        ],
                      ),
                    ),
                  )
                : SizedBox.expand(
                    child: WebViewWidget(controller: _controller!),
                  ),
          ),
        ),
        if (_isLoading)
          Container(
            color: Colors.black.withAlpha((0.1 * 255).round()),
            width: double.infinity,
            height: double.infinity,
            child: const Center(child: CircularProgressIndicator()),
          ),
        Positioned.fill(
          child: DraggableWebNavBar(
            onBack: () => _controller?.goBack(),
            onForward: () => _controller?.goForward(),
            onReload: () => _controller?.reload(),
          ),
        ),
      ],
    );
  }
}
