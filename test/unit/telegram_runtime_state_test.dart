import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/status_snapshot.dart';
import 'package:pocketclaw/src/core/telegram_runtime_state.dart';

/// PC-DEF-027. Runtime truth, never stored state. A bot username in
/// SharedPreferences cannot reach this function at all -- only Core's view can.
void main() {
  StatusChannel telegram({
    String name = 'telegram',
    bool configured = true,
    bool started = true,
    bool running = true,
  }) => StatusChannel(
    name: name,
    configured: configured,
    started: started,
    running: running,
  );

  // Case 1: the reported defect -- Core refused the channel.
  test('a channel Core refused is never Connected', () {
    final state = resolveTelegramRuntimeState([
      telegram(started: false, running: false),
    ]);
    expect(state, TelegramRuntimeState.error);
    expect(telegramMayReportConnected(state), isFalse);
  });

  // Case 2.
  test('a running channel is Connected', () {
    final state = resolveTelegramRuntimeState([telegram()]);
    expect(state, TelegramRuntimeState.running);
    expect(telegramMayReportConnected(state), isTrue);
  });

  // Cases 10 and 19: disabled, or stopped after having started.
  test('a stopped channel is configured but not running', () {
    final state = resolveTelegramRuntimeState([telegram(running: false)]);
    expect(state, TelegramRuntimeState.configuredNotRunning);
    expect(telegramMayReportConnected(state), isFalse);
  });

  test('no telegram channel means not configured', () {
    expect(
      resolveTelegramRuntimeState([]),
      TelegramRuntimeState.notConfigured,
    );
    expect(
      resolveTelegramRuntimeState([
        telegram(configured: false, started: false, running: false),
      ]),
      TelegramRuntimeState.notConfigured,
    );
  });

  // Case 18: gateway stopped, so nothing has reported.
  test('no runtime report is never Connected', () {
    final state = resolveTelegramRuntimeState(null);
    expect(state, TelegramRuntimeState.notConfigured);
    expect(telegramMayReportConnected(state), isFalse);
  });

  test('only a running channel may ever be reported as Connected', () {
    for (final state in TelegramRuntimeState.values) {
      expect(
        telegramMayReportConnected(state),
        state == TelegramRuntimeState.running,
        reason: '$state must not be presented as Connected',
      );
    }
  });

  test('channel matching is case-insensitive', () {
    expect(
      resolveTelegramRuntimeState([telegram(name: 'Telegram')]),
      TelegramRuntimeState.running,
    );
  });

  // Other channels must not be mistaken for Telegram.
  test('another running channel does not make Telegram Connected', () {
    final state = resolveTelegramRuntimeState([
      telegram(name: 'pocketclaw'),
    ]);
    expect(state, TelegramRuntimeState.notConfigured);
    expect(telegramMayReportConnected(state), isFalse);
  });
}
