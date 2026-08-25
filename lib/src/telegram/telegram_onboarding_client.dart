import 'dart:convert';
import 'dart:io';

import 'package:http/http.dart' as http;

import 'telegram_onboarding_models.dart';

/// HTTP client for PocketClaw's own Telegram onboarding service.
///
/// The base URL points at a PocketClaw-owned deployment; see
/// `services/telegram-onboarding/README.md`. No third-party onboarding
/// provider is involved.
class TelegramOnboardingClient {
  TelegramOnboardingClient({
    required this.baseUrl,
    http.Client? httpClient,
    Duration? timeout,
  })  : _http = httpClient ?? http.Client(),
        _timeout = timeout ?? const Duration(seconds: 15);

  final String baseUrl;
  final http.Client _http;
  final Duration _timeout;

  Uri _url(String path) {
    final root = baseUrl.endsWith('/')
        ? baseUrl.substring(0, baseUrl.length - 1)
        : baseUrl;
    return Uri.parse('$root$path');
  }

  /// Starts a pairing session.
  Future<TelegramPairing> createPairing() async {
    final response = await _send(
      () => _http.post(_url('/telegram/pairings')),
    );
    if (response.statusCode == 429) {
      throw const TelegramOnboardingException(
        TelegramOnboardingErrorKind.rateLimited,
      );
    }
    if (response.statusCode != 201) {
      throw TelegramOnboardingException(
        TelegramOnboardingErrorKind.serviceError,
        'create pairing returned ${response.statusCode}',
      );
    }
    return TelegramPairing.fromJson(_decode(response.body));
  }

  /// Reads the current state of a pairing. Never returns a bot token.
  Future<TelegramPairingStatus> fetchStatus(TelegramPairing pairing) async {
    final response = await _send(
      () => _http.get(
        _url('/telegram/pairings/${pairing.pairingId}'),
        headers: _authHeaders(pairing),
      ),
    );
    if (response.statusCode == 404) {
      // The service answers 404 for expired, unknown, and wrong-token alike,
      // so it cannot be used to enumerate pairings. From here all three mean
      // the same thing: this pairing can no longer complete.
      return const TelegramPairingStatus(state: PairingState.expired);
    }
    if (response.statusCode != 200) {
      throw TelegramOnboardingException(
        TelegramOnboardingErrorKind.serviceError,
        'poll returned ${response.statusCode}',
      );
    }
    return TelegramPairingStatus.fromJson(_decode(response.body));
  }

  /// Collects the child bot token. The service allows this exactly once.
  Future<TelegramBotCredentials> collectCredentials(
    TelegramPairing pairing,
  ) async {
    final response = await _send(
      () => _http.post(
        _url('/telegram/pairings/${pairing.pairingId}/token'),
        headers: _authHeaders(pairing),
      ),
    );
    if (response.statusCode == 404 || response.statusCode == 409) {
      throw const TelegramOnboardingException(
        TelegramOnboardingErrorKind.pairingGone,
      );
    }
    if (response.statusCode != 200) {
      throw TelegramOnboardingException(
        TelegramOnboardingErrorKind.serviceError,
        'token collection returned ${response.statusCode}',
      );
    }
    return TelegramBotCredentials.fromJson(_decode(response.body));
  }

  Map<String, String> _authHeaders(TelegramPairing pairing) => {
        'Authorization': 'Bearer ${pairing.pollToken}',
      };

  Future<http.Response> _send(Future<http.Response> Function() request) async {
    try {
      return await request().timeout(_timeout);
    } on TelegramOnboardingException {
      rethrow;
    } on SocketException {
      throw const TelegramOnboardingException(
        TelegramOnboardingErrorKind.network,
      );
    } catch (error) {
      // Timeouts, malformed URLs, and TLS failures all land here. The detail
      // is kept generic: a transport error can quote the request, and no
      // request detail is worth risking in a user-visible string.
      throw const TelegramOnboardingException(
        TelegramOnboardingErrorKind.network,
      );
    }
  }

  Map<String, dynamic> _decode(String body) {
    try {
      final decoded = jsonDecode(body);
      if (decoded is! Map<String, dynamic>) {
        throw const FormatException('expected a JSON object');
      }
      return decoded;
    } catch (_) {
      throw const TelegramOnboardingException(
        TelegramOnboardingErrorKind.serviceError,
        'malformed response',
      );
    }
  }

  void close() => _http.close();
}
