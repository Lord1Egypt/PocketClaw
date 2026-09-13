import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/status_snapshot.dart';
import 'package:pocketclaw/src/core/telegram_runtime_state.dart';

/// PC-DEF-027. Configuration truth and runtime truth are separate, and each
/// state is claimed only when something actually says so.
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

  TelegramRuntimeState resolve({
    bool configuredAndValid = true,
    List<StatusChannel>? channels,
    String? startupError,
  }) => resolveTelegramRuntimeState(
    configuredAndValid: configuredAndValid,
    channels: channels,
    startupError: startupError,
  );

  group('running', () {
    test('a running channel is Connected', () {
      final state = resolve(channels: [telegram()]);
      expect(state, TelegramRuntimeState.running);
      expect(telegramMayReportConnected(state), isTrue);
    });

    test('running outranks a configuration that no longer validates', () {
      // Edited config that has not been applied yet must not make a live
      // channel disappear from the UI.
      expect(
        resolve(configuredAndValid: false, channels: [telegram()]),
        TelegramRuntimeState.running,
      );
    });
  });

  group('configured but the runtime is not active', () {
    // The correction: a stopped Gateway must never read as "not configured".
    test('configured with the Gateway stopped stays configured', () {
      final state = resolve(configuredAndValid: true, channels: null);
      expect(state, TelegramRuntimeState.configuredRuntimeNotActive);
      expect(telegramMayReportConnected(state), isFalse);
      expect(state, isNot(TelegramRuntimeState.notConfigured));
      expect(state, isNot(TelegramRuntimeState.error));
    });

    test('configured with no snapshot yet stays configured', () {
      expect(
        resolve(configuredAndValid: true, channels: null),
        TelegramRuntimeState.configuredRuntimeNotActive,
      );
    });

    test('configured but the runtime has not loaded the channel', () {
      expect(
        resolve(configuredAndValid: true, channels: []),
        TelegramRuntimeState.configuredRuntimeNotActive,
      );
    });

    // The second correction: not started is not the same as failed.
    test('not started is not an error', () {
      final state = resolve(
        channels: [telegram(started: false, running: false)],
      );
      expect(
        state,
        TelegramRuntimeState.configuredRuntimeNotActive,
        reason: 'ERROR needs affirmative evidence, not the absence of a start',
      );
      expect(telegramMayReportConnected(state), isFalse);
    });

    test('a stopped channel that had started is not an error either', () {
      expect(
        resolve(channels: [telegram(running: false)]),
        TelegramRuntimeState.configuredRuntimeNotActive,
      );
    });

    test('a disabled channel is not Running and not an error', () {
      final state = resolve(
        channels: [telegram(configured: false, started: false, running: false)],
      );
      expect(state, TelegramRuntimeState.configuredRuntimeNotActive);
      expect(telegramMayReportConnected(state), isFalse);
    });
  });

  group('not configured', () {
    test('no configuration and no runtime', () {
      expect(
        resolve(configuredAndValid: false, channels: null),
        TelegramRuntimeState.notConfigured,
      );
    });

    test('no configuration while the runtime reports nothing running', () {
      expect(
        resolve(configuredAndValid: false, channels: []),
        TelegramRuntimeState.notConfigured,
      );
    });

    test('another running channel is not Telegram', () {
      expect(
        resolve(configuredAndValid: false, channels: [telegram(name: 'pocketclaw')]),
        TelegramRuntimeState.notConfigured,
      );
    });
  });

  group('error', () {
    test('only affirmative failure evidence produces an error', () {
      final state = resolve(startupError: 'channel failed to start');
      expect(state, TelegramRuntimeState.error);
      expect(telegramMayReportConnected(state), isFalse);
    });

    test('blank or absent failure evidence is not an error', () {
      expect(resolve(startupError: null), TelegramRuntimeState.configuredRuntimeNotActive);
      expect(resolve(startupError: '   '), TelegramRuntimeState.configuredRuntimeNotActive);
    });

    test('a running channel is never an error', () {
      expect(
        resolve(channels: [telegram()], startupError: 'stale failure'),
        TelegramRuntimeState.running,
      );
    });
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
      resolve(channels: [telegram(name: 'Telegram')]),
      TelegramRuntimeState.running,
    );
  });

  // The Gateway-stopped lifecycle the correction asked to be proven.
  test('configured, Gateway stopped, then started: configured throughout', () {
    final stopped = resolve(configuredAndValid: true, channels: null);
    expect(stopped, TelegramRuntimeState.configuredRuntimeNotActive);
    expect(telegramMayReportConnected(stopped), isFalse);

    final starting = resolve(
      configuredAndValid: true,
      channels: [telegram(started: false, running: false)],
    );
    expect(starting, TelegramRuntimeState.configuredRuntimeNotActive);

    final up = resolve(configuredAndValid: true, channels: [telegram()]);
    expect(up, TelegramRuntimeState.running);
    expect(telegramMayReportConnected(up), isTrue);
  });
}
