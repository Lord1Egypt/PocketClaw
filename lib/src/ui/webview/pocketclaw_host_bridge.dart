import 'dart:convert';

/// What the embedded Core console asked the Flutter host to do.
enum HostRequestKind {
  /// Channels → Telegram wants the native managed-bot flow.
  openTelegramOnboarding,

  /// A link that must leave the WebView, such as the bot's `t.me` chat.
  openExternal,

  /// Channels → WhatsApp Self-Chat wants WhatsApp opened on the user's own
  /// number with a message prepared. Never sent.
  openWhatsAppSelfChat,
}

class HostRequest {
  const HostRequest(this.kind, {this.url, this.selfNumber, this.message});

  final HostRequestKind kind;
  final String? url;

  /// The user's own WhatsApp number, in canonical international form.
  final String? selfNumber;

  /// The body to place in WhatsApp's compose box. Never logged.
  final String? message;
}

/// The contract between the embedded Core console and the Flutter host.
///
/// The console is a React app served by Core inside the WebView; the pairing
/// flow is Dart. Rather than reimplement pairing in TypeScript, the console
/// renders the entry point and asks the host to run the flow it already has.
abstract final class PocketClawHostBridge {
  /// The `addJavaScriptChannel` name the injected script posts to.
  static const String channelName = 'PocketClawHost';

  /// Injected after every page load, and again whenever the Telegram
  /// configuration changes, so the console can render the right surface.
  ///
  /// Carries no secret: whether an onboarding endpoint was compiled in, and
  /// the paired bot's public `@username`. Never the bot token.
  static String bootstrapScript({
    required bool onboardingConfigured,
    String? telegramBotUsername,
  }) {
    final username = telegramBotUsername == null || telegramBotUsername.isEmpty
        ? 'null'
        : jsonEncode(telegramBotUsername);
    return '''
(function () {
  window.__pocketclawHost = {
    platform: 'android',
    onboardingConfigured: $onboardingConfigured,
    telegramBotUsername: $username,
    openTelegramOnboarding: function () {
      $channelName.postMessage(JSON.stringify({ type: 'openTelegramOnboarding' }));
    },
    openExternal: function (url) {
      $channelName.postMessage(JSON.stringify({ type: 'openExternal', url: url }));
    },
    openWhatsAppSelfChat: function (selfNumber, message) {
      $channelName.postMessage(JSON.stringify({
        type: 'openWhatsAppSelfChat',
        selfNumber: selfNumber,
        message: message
      }));
    }
  };
  window.dispatchEvent(new CustomEvent('pocketclaw:host-ready'));
})();
''';
  }

  /// Asks the page whether it is still alive and rendering.
  ///
  /// This exists because Android may kill a backgrounded WebView's renderer
  /// process to reclaim memory. When it does the view keeps its layout but
  /// shows nothing, and `webview_flutter_android` exposes no
  /// `onRenderProcessGone` callback, so Dart is never told. Asking the page a
  /// question it can only answer while alive is the available signal.
  ///
  /// Returns `alive` only when the console has mounted and its root still has
  /// content. Anything else — a thrown error, a null result, or a page that
  /// never runs the script at all — means the page is not usable. Which of
  /// those happened is recorded; why it happened is not inferred.
  static const String livenessProbeScript = r"""
(function () {
  try {
    if (!window.__pocketclawReady) return 'not-ready';
    var root = document.getElementById('root');
    if (!root || root.childElementCount === 0) return 'empty';
    return 'alive';
  } catch (e) {
    return 'error';
  }
})();
""";

  /// Reads the console's current in-app route so a recovery reload can return
  /// the user where they were rather than to the home page.
  static const String currentRouteScript = r"""
(function () {
  try {
    return window.location.pathname + window.location.search;
  } catch (e) {
    return '';
  }
})();
""";

  /// Whether a liveness probe result means the page is usable.
  ///
  /// A page that cannot run script produces an error rather than a value, and
  /// `runJavaScriptReturningResult` returns platform-shaped values, so anything
  /// that is not an explicit `alive` is treated as unusable.
  static bool isAliveResult(Object? result) {
    if (result == null) return false;
    final text = result.toString().replaceAll('"', '').trim();
    return text == 'alive';
  }

  /// Normalises a probe result for a route string.
  static String? routeFromResult(Object? result) {
    if (result == null) return null;
    final text = result.toString().replaceAll('"', '').trim();
    if (text.isEmpty || !text.startsWith('/')) return null;
    return text;
  }

  /// Fired after the native flow returns so the console reloads the channel
  /// configuration instead of showing a stale disconnected state.
  static const String telegramUpdatedScript =
      "window.dispatchEvent(new CustomEvent('pocketclaw:telegram-updated'));";

  /// Whether [loadedUrl] is the same origin as the local Core console at
  /// [consoleUrl].
  ///
  /// The host contract is published to, and accepted from, the console only.
  /// The WebView follows outbound links if a user taps one, and a third-party
  /// page holding `openTelegramOnboarding` and `openExternal` could drive the
  /// app, so both directions are scoped by origin. Unparseable input fails
  /// closed.
  static bool isSameOrigin(String? loadedUrl, String consoleUrl) {
    if (loadedUrl == null) return false;
    final page = Uri.tryParse(loadedUrl);
    final console = Uri.tryParse(consoleUrl);
    if (page == null || console == null) return false;
    if (!page.hasAuthority || !console.hasAuthority) return false;
    return page.scheme == console.scheme &&
        page.host == console.host &&
        page.port == console.port;
  }

  /// Parses one message from the console.
  ///
  /// The WebView renders a locally served page, but this is still a string
  /// crossing a trust boundary, so anything unrecognised is dropped rather
  /// than acted on. [HostRequestKind.openExternal] is only produced for an
  /// absolute http(s) URL — never for a `javascript:`, `file:` or `intent:`
  /// scheme that would turn the bridge into an arbitrary-launch primitive.
  static HostRequest? parseMessage(String raw) {
    final Object? decoded;
    try {
      decoded = jsonDecode(raw);
    } on FormatException {
      return null;
    }
    if (decoded is! Map) return null;

    switch (decoded['type']) {
      case 'openTelegramOnboarding':
        return const HostRequest(HostRequestKind.openTelegramOnboarding);
      case 'openExternal':
        final url = decoded['url'];
        if (url is! String) return null;
        final uri = Uri.tryParse(url);
        if (uri == null || !uri.hasScheme || !uri.hasAuthority) return null;
        if (uri.scheme != 'http' && uri.scheme != 'https') return null;
        return HostRequest(HostRequestKind.openExternal, url: url);
      case 'openWhatsAppSelfChat':
        final selfNumber = decoded['selfNumber'];
        final message = decoded['message'];
        if (selfNumber is! String || message is! String) return null;
        if (selfNumber.trim().isEmpty || message.trim().isEmpty) return null;
        return HostRequest(
          HostRequestKind.openWhatsAppSelfChat,
          selfNumber: selfNumber.trim(),
          message: message,
        );
      default:
        return null;
    }
  }
}
