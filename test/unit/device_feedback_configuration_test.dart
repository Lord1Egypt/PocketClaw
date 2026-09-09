import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/device_feedback_models.dart';
import 'package:pocketclaw/src/core/service_manager.dart';

/// Optional device feedback, after H1.5 removed Firebase.
///
/// The property that matters is not which providers exist but that feedback is
/// off unless someone explicitly configured it. Firebase used to be the
/// fall-through for an unrecognised value, which meant a build that simply did
/// not pass the dart-define selected an analytics provider by accident.
void main() {
  group('provider selection', () {
    test('an unrecognised or absent value means no provider', () {
      for (final raw in ['', '   ', 'firebase', 'nonsense', 'off', 'disabled']) {
        expect(DeviceFeedbackProvider.fromEnvironmentValue(raw),
            DeviceFeedbackProvider.none,
            reason: '"$raw" must not select a provider');
      }
    });

    test('Firebase is no longer selectable at all', () {
      // Not merely disabled: the value does not resolve to anything, because
      // the SDK it named is gone from the build.
      expect(DeviceFeedbackProvider.values,
          isNot(contains(anyOf(equals('firebase')))));
      expect(DeviceFeedbackProvider.values.map((v) => v.name).toList(),
          ['none', 'umeng']);
    });

    test('Umeng is selected only when explicitly named', () {
      expect(DeviceFeedbackProvider.fromEnvironmentValue('umeng'),
          DeviceFeedbackProvider.umeng);
      expect(DeviceFeedbackProvider.fromEnvironmentValue('UMENG'),
          DeviceFeedbackProvider.umeng);
    });
  });

  group('configuration resolution', () {
    test('a requested provider without its key stays off', () {
      expect(
        ServiceManager.resolveDeviceFeedbackProvider(
          requested: DeviceFeedbackProvider.umeng,
          umengAppKey: '',
        ),
        DeviceFeedbackProvider.none,
      );
    });

    test('a requested provider with its key is honoured', () {
      expect(
        ServiceManager.resolveDeviceFeedbackProvider(
          requested: DeviceFeedbackProvider.umeng,
          umengAppKey: 'app-key',
        ),
        DeviceFeedbackProvider.umeng,
      );
    });

    test('nothing requested stays off regardless of what is configured', () {
      expect(
        ServiceManager.resolveDeviceFeedbackProvider(
          requested: DeviceFeedbackProvider.none,
          umengAppKey: 'app-key',
        ),
        DeviceFeedbackProvider.none,
      );
    });
  });
}
