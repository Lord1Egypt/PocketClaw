import 'dart:convert';

/// What the embedded Core console asked the Flutter host to do.
enum HostRequestKind {
  /// Channels → Telegram wants the native managed-bot flow.
  openTelegramOnboarding,

  /// A link that must leave the WebView, such as the bot's `t.me` chat.
  openExternal,
}

class HostRequest {
  const HostRequest(this.kind, {this.url});

  final HostRequestKind kind;
  final String? url;
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
    }
  };
  window.dispatchEvent(new CustomEvent('pocketclaw:host-ready'));
})();
''';
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
      default:
        return null;
    }
  }
}
