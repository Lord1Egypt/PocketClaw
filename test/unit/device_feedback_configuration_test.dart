import 'package:flutter_test/flutter_test.dart';
import 'package:picoclaw_flutter_ui/src/core/device_feedback_models.dart';
import 'package:picoclaw_flutter_ui/src/core/service_manager.dart';

void main() {
  group('optional device feedback configuration', () {
    test('disables Firebase when any required build value is absent', () {
      final provider = ServiceManager.resolveDeviceFeedbackProvider(
        requested: DeviceFeedbackProvider.firebase,
        firebaseProjectId: 'project',
        firebaseApiKey: '',
        firebaseAppId: 'app',
        firebaseMessagingSenderId: 'sender',
        umengAppKey: '',
      );

      expect(provider, DeviceFeedbackProvider.none);
    });

    test('enables Firebase only with complete explicit configuration', () {
      final provider = ServiceManager.resolveDeviceFeedbackProvider(
        requested: DeviceFeedbackProvider.firebase,
        firebaseProjectId: 'project',
        firebaseApiKey: 'api-key',
        firebaseAppId: 'app',
        firebaseMessagingSenderId: 'sender',
        umengAppKey: '',
      );

      expect(provider, DeviceFeedbackProvider.firebase);
    });

    test('keeps feedback disabled when no provider was requested', () {
      final provider = ServiceManager.resolveDeviceFeedbackProvider(
        requested: DeviceFeedbackProvider.none,
        firebaseProjectId: '',
        firebaseApiKey: '',
        firebaseAppId: '',
        firebaseMessagingSenderId: '',
        umengAppKey: '',
      );

      expect(provider, DeviceFeedbackProvider.none);
    });
  });
}
