import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/autostart_coordinator.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.lord1egypt.pocketclaw/picoclaw');
  late ServiceManager service;
  var gatewayStatus = 'stopped';

  setUp(() {
    SharedPreferences.setMockInitialValues({});
    service = ServiceManager();
    gatewayStatus = 'stopped';
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          switch (call.method) {
            case 'getServiceStatus':
              return <String, Object?>{
                'isRunning': true,
                'isStarting': false,
                'isStopping': false,
                'manualStopActive': false,
                'pid': 1234,
                'lastStartOperationId': 'manual-service-1',
              };
            case 'checkHealth':
              return <String, Object?>{
                'isHealthy': true,
                'status': 'ok',
                'pid': 1234,
              };
            case 'getGatewayStatus':
              return <String, Object?>{'gateway_status': gatewayStatus};
            case 'takeNewLogs':
              return <String>[];
            default:
              return null;
          }
        });
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test(
    'manual Gateway start and stop are reflected by the live native poll',
    () async {
      gatewayStatus = 'running';
      await service.pollNativeServiceStatusForTest(includeGateway: true);
      expect(service.status, ServiceStatus.running);
      expect(service.gatewayStatus, AutoStartRuntimeState.running);

      gatewayStatus = 'stopped';
      await service.pollNativeServiceStatusForTest(includeGateway: true);
      expect(service.gatewayStatus, AutoStartRuntimeState.stopped);
    },
  );

  test(
    'canonical RUNNING clears stale Service and Gateway startup errors',
    () async {
      service.setRuntimeFailureForTest(
        serviceError: 'PocketClaw service startup timed out. Try again.',
        gatewayError: 'Gateway startup timed out. Try again.',
      );
      gatewayStatus = 'running';

      await service.pollNativeServiceStatusForTest(includeGateway: true);

      expect(service.status, ServiceStatus.running);
      expect(service.gatewayStatus, AutoStartRuntimeState.running);
      expect(service.serviceStartError, isNull);
      expect(service.gatewayStartError, isNull);
    },
  );

  test('an older operation generation cannot overwrite a newer operation', () {
    expect(
      ServiceManager.isCurrentOperationGenerationForTest(
        operationGeneration: 4,
        currentGeneration: 5,
      ),
      isFalse,
    );
    expect(
      ServiceManager.isCurrentOperationGenerationForTest(
        operationGeneration: 5,
        currentGeneration: 5,
      ),
      isTrue,
    );
  });
}
