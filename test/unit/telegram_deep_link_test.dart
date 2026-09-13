import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'package:pocketclaw/src/telegram/telegram_deep_link.dart';

/// PC-DEF-052. The hosting origin must never be part of the user's navigation.
/// The app resolves the setup link in the background and opens Telegram, or it
/// opens nothing at all.

/// A client that answers with redirects from a fixed map, and records what was
/// requested so a test can prove no extra hop was made.
class _RedirectClient extends http.BaseClient {
  _RedirectClient(this.locations);

  final Map<String, String?> locations;
  final List<String> requested = [];
  final List<Map<String, String>> sentHeaders = [];

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    requested.add(request.url.toString());
    sentHeaders.add(Map<String, String>.from(request.headers));
    final location = locations[request.url.toString()];
    if (location == null) {
      return http.StreamedResponse(
        Stream.value(utf8.encode('done')),
        200,
        request: request,
      );
    }
    return http.StreamedResponse(
      Stream.value(const <int>[]),
      302,
      headers: {'location': location},
      request: request,
    );
  }
}

void main() {
  group('TelegramDeepLink.isTelegramTarget', () {
    test('accepts Telegram web and app links', () {
      expect(TelegramDeepLink.isTelegramTarget('https://t.me/newbot/Mgr/x_bot'),
          isTrue);
      expect(TelegramDeepLink.isTelegramTarget('https://telegram.me/x'), isTrue);
      expect(TelegramDeepLink.isTelegramTarget('tg://resolve?domain=x'), isTrue);
    });

    test('rejects the hosting origin and anything else', () {
      for (final url in [
        'https://pocketclaw-telegram-setup.vercel.app/go/abc',
        'https://example.com/t.me/newbot',
        'http://t.me/newbot/Mgr/x_bot', // plaintext
        'https://evil.invalid/?next=https://t.me/x',
        'javascript:alert(1)',
        'https://t.me.evil.invalid/x',
        '',
      ]) {
        expect(TelegramDeepLink.isTelegramTarget(url), isFalse, reason: url);
      }
    });
  });

  group('TelegramDeepLinkResolver', () {
    test('passes a Telegram link through without making a request', () async {
      final client = _RedirectClient({});
      final resolver = TelegramDeepLinkResolver(httpClient: client);

      const link = 'https://t.me/newbot/Mgr/pocketclaw_abc_bot?name=Agent';
      expect(await resolver.resolve(link), link);
      expect(client.requested, isEmpty,
          reason: 'a link that is already Telegram needs no network round trip');
    });

    // The reported flow: the service issues its own redirect URL.
    test('resolves a hosting redirect to the Telegram link behind it', () async {
      const hosted = 'https://pocketclaw-telegram-setup.vercel.app/go/abc';
      const telegram = 'https://t.me/newbot/Mgr/pocketclaw_abc_bot';
      final client = _RedirectClient({hosted: telegram});
      final resolver = TelegramDeepLinkResolver(httpClient: client);

      expect(await resolver.resolve(hosted), telegram);
      expect(client.requested, [hosted]);
    });

    test('follows a relative Location against the current URL', () async {
      const hosted = 'https://host.invalid/go/abc';
      final client = _RedirectClient({
        hosted: '/next',
        'https://host.invalid/next': 'https://t.me/newbot/Mgr/x_bot',
      });
      final resolver = TelegramDeepLinkResolver(httpClient: client);

      expect(await resolver.resolve(hosted), 'https://t.me/newbot/Mgr/x_bot');
    });

    test('refuses a redirect chain that never reaches Telegram', () async {
      const hosted = 'https://host.invalid/go/abc';
      final client = _RedirectClient({hosted: 'https://elsewhere.invalid/page'});
      final resolver = TelegramDeepLinkResolver(httpClient: client);

      expect(await resolver.resolve(hosted), isNull,
          reason: 'returning the non-Telegram URL is the defect');
    });

    test('refuses a redirect that downgrades to plaintext', () async {
      const hosted = 'https://host.invalid/go/abc';
      final client = _RedirectClient({hosted: 'http://t.me/newbot/Mgr/x_bot'});
      final resolver = TelegramDeepLinkResolver(httpClient: client);

      expect(await resolver.resolve(hosted), isNull);
    });

    test('refuses a non-https starting URL outright', () async {
      final client = _RedirectClient({});
      final resolver = TelegramDeepLinkResolver(httpClient: client);

      expect(await resolver.resolve('http://host.invalid/go/abc'), isNull);
      expect(await resolver.resolve('file:///etc/passwd'), isNull);
      expect(client.requested, isEmpty,
          reason: 'a scheme that cannot be a setup link is never dereferenced');
    });

    test('gives up rather than following an unbounded redirect loop', () async {
      const a = 'https://host.invalid/a';
      final client = _RedirectClient({a: a});
      final resolver = TelegramDeepLinkResolver(httpClient: client);

      expect(await resolver.resolve(a), isNull);
      expect(client.requested.length, TelegramDeepLink.maxRedirects);
    });

    test('returns null when the endpoint does not redirect at all', () async {
      const hosted = 'https://host.invalid/go/abc';
      final client = _RedirectClient({});
      final resolver = TelegramDeepLinkResolver(httpClient: client);

      expect(await resolver.resolve(hosted), isNull);
    });

    // Nothing is authenticated to a host that has not been vetted yet, and the
    // setup link is public by construction.
    test('sends no credential while resolving', () async {
      const hosted = 'https://host.invalid/go/abc';
      final client = _RedirectClient({hosted: 'https://t.me/newbot/Mgr/x_bot'});
      final resolver = TelegramDeepLinkResolver(httpClient: client);

      await resolver.resolve(hosted);

      for (final headers in client.sentHeaders) {
        final keys = headers.keys.map((k) => k.toLowerCase()).toList();
        expect(keys, isNot(contains('authorization')));
        expect(keys, isNot(contains('cookie')));
      }
    });

    test('returns null on a transport failure instead of throwing', () async {
      final resolver = TelegramDeepLinkResolver(httpClient: _ThrowingClient());

      expect(await resolver.resolve('https://host.invalid/go/abc'), isNull);
    });

    test('returns null on an empty or blank link', () async {
      final resolver = TelegramDeepLinkResolver(httpClient: _RedirectClient({}));

      expect(await resolver.resolve(''), isNull);
      expect(await resolver.resolve('   '), isNull);
    });
  });
}

class _ThrowingClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    throw const SocketExceptionLike();
  }
}

class SocketExceptionLike implements Exception {
  const SocketExceptionLike();
}
