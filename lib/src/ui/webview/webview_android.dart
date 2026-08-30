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

class _WebViewAndroidState extends State<WebViewAndroid>
    with WidgetsBindingObserver {
  WebViewController? _controller;
  bool _isLoading = true;
  String? _telegramBotUsername;
  String? _loadedUrl;

  /// Guards against a recovery loop.
  ///
  /// A page that reloads and is still not alive must not be reloaded again on
  /// the next resume tick, or a genuinely broken console would flicker forever
  /// instead of leaving the user the Refresh button.
  bool _recovering = false;
  int _consecutiveFailedRecoveries = 0;
  static const int _maxConsecutiveRecoveries = 1;

  /// Android can take a moment to hand the renderer back after resume. Probing
  /// instantly would report a healthy page as dead and reload it needlessly.
  static const Duration _resumeProbeDelay = Duration(milliseconds: 400);

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
        final username = await TelegramOnboardingLauncher.startPairing(context);
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
    WidgetsBinding.instance.addObserver(this);
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
            _logLifecycle('webview.page.started');
            if (mounted) setState(() => _isLoading = true);
          },
          onProgress: (progress) {
            if (progress < 100 && mounted) setState(() => _isLoading = true);
          },
          onPageFinished: (url) {
            _loadedUrl = url;
            _logLifecycle('webview.page.finished');
            // A page that loaded is a page that came back; let a later resume
            // recover again if it dies once more.
            _consecutiveFailedRecoveries = 0;
            unawaited(_injectHost());
            if (mounted) setState(() => _isLoading = false);
          },
          onWebResourceError: (error) {
            // The gateway is briefly unreachable during a configuration
            // restart. That is expected and must not trigger recovery on its
            // own; only a resume that finds a dead page does.
            _logLifecycle('webview.resource.error', {
              'main_frame': error.isForMainFrame ?? false,
            });
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
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    super.didChangeAppLifecycleState(state);
    switch (state) {
      case AppLifecycleState.paused:
      case AppLifecycleState.hidden:
        _logLifecycle('app.background');
      case AppLifecycleState.resumed:
        _logLifecycle('app.foreground');
        unawaited(_recoverIfPageIsDead());
      case AppLifecycleState.inactive:
      case AppLifecycleState.detached:
        break;
    }
  }

  void _logLifecycle(String event, [Map<String, Object?> fields = const {}]) {
    final detail = fields.isEmpty
        ? ''
        : ' ${fields.entries.map((e) => '${e.key}=${e.value}').join(' ')}';
    // Page contents and URLs beyond the console origin are never logged.
    debugPrint('[webview] $event$detail');
  }

  /// Restores the console after a resume that left it blank, and only then.
  ///
  /// Android may kill a backgrounded WebView's renderer process to reclaim
  /// memory. The view keeps its layout but renders nothing, and the plugin
  /// exposes no renderer-gone callback, so the only way to notice is to ask the
  /// page whether it is still there.
  ///
  /// A healthy page is left completely alone: reloading on every resume would
  /// throw away scroll position and page state, and would hide this defect
  /// rather than fix it.
  Future<void> _recoverIfPageIsDead() async {
    final controller = _controller;
    if (controller == null || _recovering) return;

    _logLifecycle('webview.resume');
    await Future<void>.delayed(_resumeProbeDelay);
    if (!mounted) return;

    Object? probe;
    var probeFailed = false;
    try {
      probe = await controller.runJavaScriptReturningResult(
        PocketClawHostBridge.livenessProbeScript,
      );
    } catch (_) {
      // A dead renderer cannot run script at all; the throw is the answer.
      probeFailed = true;
    }

    if (!probeFailed && PocketClawHostBridge.isAliveResult(probe)) {
      _consecutiveFailedRecoveries = 0;
      _logLifecycle('frontend.resume.health_check', {'result': 'alive'});
      return;
    }

    _logLifecycle('frontend.resume.health_check', {
      'result': probeFailed ? 'renderer_gone' : 'blank',
    });

    if (_consecutiveFailedRecoveries >= _maxConsecutiveRecoveries) {
      // Recovery already ran and did not help. Leave the page as it is so the
      // Refresh control stays useful instead of fighting it.
      _logLifecycle('frontend.resume.recovery.failed', {'reason': 'exhausted'});
      return;
    }

    await _restoreLostPage(controller);
  }

  Future<void> _restoreLostPage(WebViewController controller) async {
    _recovering = true;
    _consecutiveFailedRecoveries++;
    _logLifecycle('frontend.resume.recovery.started');

    // Reload the route the user was on rather than the console's home page. A
    // dead renderer cannot report its route, so the last URL observed through
    // onUrlChange is the fallback.
    var target = _loadedUrl ?? widget.url;
    try {
      final route = PocketClawHostBridge.routeFromResult(
        await controller.runJavaScriptReturningResult(
          PocketClawHostBridge.currentRouteScript,
        ),
      );
      if (route != null) {
        target = Uri.parse(widget.url).replace(path: route).toString();
      }
    } catch (_) {
      // Expected when the renderer is gone; the observed URL still applies.
    }

    try {
      await controller.loadRequest(Uri.parse(target));
      _logLifecycle('frontend.resume.recovery.completed');
    } catch (e) {
      _logLifecycle('frontend.resume.recovery.failed', {'stage': 'load'});
    } finally {
      _recovering = false;
    }
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
