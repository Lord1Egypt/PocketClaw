import 'dart:async';

import 'package:http/http.dart' as http;

/// Where PocketClaw is allowed to send the user during Telegram onboarding.
///
/// PC-DEF-052. The onboarding service issues the setup link, and a deployment
/// that hosts it behind its own redirect endpoint made the hosting origin part
/// of the user's navigation: the browser opened, showed `*.vercel.app` for a
/// moment, and then redirected into Telegram. The intermediate page does nothing
/// for the user and names infrastructure that is not theirs to think about.
///
/// So the app stops opening whatever URL it is handed. A link is either already a
/// Telegram link, or it is resolved to one in the background before anything is
/// shown. Anything that does not resolve to Telegram is refused rather than
/// opened, which also makes this the boundary that stops a compromised or
/// misconfigured service from steering the user somewhere arbitrary — a network
/// response is untrusted input, and this one is fed straight to the OS.
abstract final class TelegramDeepLink {
  /// Telegram's own web host and its app scheme. Nothing else is a destination.
  static const Set<String> _telegramHosts = {'t.me', 'telegram.me', 'telegram.dog'};
  static const String _telegramScheme = 'tg';

  /// How many redirect hops to follow. A setup link needs one; more than a few
  /// means the endpoint is not a simple redirector and is not worth chasing.
  static const int maxRedirects = 5;

  /// Whether [url] is somewhere PocketClaw may send the user for onboarding.
  static bool isTelegramTarget(String url) {
    final uri = Uri.tryParse(url.trim());
    if (uri == null) return false;
    if (uri.isScheme(_telegramScheme)) return true;
    if (!uri.isScheme('https')) return false;
    return _telegramHosts.contains(uri.host.toLowerCase());
  }
}

/// Turns the service's setup link into a Telegram link, without showing the user
/// anything in between.
class TelegramDeepLinkResolver {
  TelegramDeepLinkResolver({http.Client? httpClient, Duration? timeout})
      : _http = httpClient ?? http.Client(),
        _timeout = timeout ?? const Duration(seconds: 10);

  final http.Client _http;
  final Duration _timeout;

  /// Returns the Telegram URL to open, or null when none could be established.
  ///
  /// Null means "do not navigate". The caller reports that onboarding could not
  /// start; it must not fall back to opening [rawUrl], because opening it is the
  /// defect.
  Future<String?> resolve(String rawUrl) async {
    final trimmed = rawUrl.trim();
    if (trimmed.isEmpty) return null;

    // Already a Telegram link: nothing to resolve, and no request to make.
    if (TelegramDeepLink.isTelegramTarget(trimmed)) return trimmed;

    var current = Uri.tryParse(trimmed);
    // Only an https redirector is followed. Refusing http keeps the hop off a
    // plaintext connection, and refusing every other scheme keeps this from
    // being a way to make the app dereference something unexpected.
    if (current == null || !current.isScheme('https')) return null;

    for (var hop = 0; hop < TelegramDeepLink.maxRedirects; hop++) {
      final location = await _redirectTarget(current!);
      if (location == null) return null;

      final next = current.resolve(location);
      if (TelegramDeepLink.isTelegramTarget(next.toString())) {
        return next.toString();
      }
      if (!next.isScheme('https')) return null;
      current = next;
    }
    return null;
  }

  /// Issues one request that is deliberately not followed, and reads where the
  /// service wanted to send the user.
  ///
  /// No credential is attached. The setup link is public by construction — it
  /// names the manager bot and the suggested username, both of which Telegram is
  /// about to show the user — so there is nothing to authenticate with and
  /// nothing that should be sent to a host this method has not yet vetted.
  Future<String?> _redirectTarget(Uri url) async {
    try {
      final request = http.Request('GET', url)
        ..followRedirects = false
        ..persistentConnection = false;
      final response = await _http.send(request).timeout(_timeout);
      // Drain, so the connection is not left half-read.
      await response.stream.drain<void>();

      if (response.statusCode < 300 || response.statusCode >= 400) return null;
      final location = response.headers['location'];
      if (location == null || location.trim().isEmpty) return null;
      return location.trim();
    } on TimeoutException {
      return null;
    } catch (_) {
      // A transport failure, a malformed Location, a TLS error. None of them
      // produce a destination, and none is worth reporting in detail.
      return null;
    }
  }

  void close() => _http.close();
}

/// Telegram's username shape, applied strictly.
///
/// The job is not to be generous about what a username may look like; it is to
/// make sure a display string can never become a destination. This rejects a
/// second `@`, a path separator, a scheme, a dot, a space, and every
/// bidirectional control character.
final RegExp _telegramUsername = RegExp(r'^[A-Za-z][A-Za-z0-9_]{3,30}[A-Za-z0-9]$');

/// Trims, removes **exactly one** optional leading `@`, and validates.
///
/// PC-DEF-075. Telegram's canonical bot link is `https://t.me/name`;
/// `https://t.me/@name` is a different, non-existent username and Telegram
/// answers "Username not found". The value reaching here comes from the
/// onboarding service, and this package could not agree on its shape — one
/// doc comment promised an `@username`, another formatter added the `@` itself.
/// So it is canonicalised rather than trusted.
///
/// Exactly one `@` is removed: stripping repeatedly would quietly turn `@@name`
/// — already evidence that something upstream is wrong — into a working link.
///
/// Returns null when the value is not a usable Telegram username, so a caller
/// cannot build a link from junk.
String? canonicalTelegramUsername(String? raw) {
  if (raw == null) return null;
  var value = raw.trim();
  if (value.startsWith('@')) value = value.substring(1);
  value = value.trim();
  return _telegramUsername.hasMatch(value) ? value : null;
}

/// The canonical bot chat link, or null when there is no usable username.
///
/// Null means "offer no link", never "offer a broken one".
String? telegramBotChatUrl(String? raw) {
  final username = canonicalTelegramUsername(raw);
  return username == null ? null : 'https://t.me/$username';
}
