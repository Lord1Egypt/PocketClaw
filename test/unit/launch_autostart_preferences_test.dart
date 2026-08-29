import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/launch_autostart_preferences.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues({});
  });

  test(
    'defaults persist across store recreation and app resume load',
    () async {
      final first = _flutterStore();
      final defaults = await first.load();
      expect(defaults.preferences.serviceEnabled, isTrue);
      expect(defaults.preferences.gatewayEnabled, isTrue);

      await first.update(serviceEnabled: false, gatewayEnabled: true);
      final recreated = _flutterStore();
      final resumed = await recreated.load();
      expect(resumed.preferences.serviceEnabled, isFalse);
      expect(resumed.preferences.gatewayEnabled, isTrue);
    },
  );

  test(
    'existing Flutter values migrate once into native canonical store',
    () async {
      SharedPreferences.setMockInitialValues({
        LaunchAutoStartPreferenceStore.serviceKey: true,
        LaunchAutoStartPreferenceStore.gatewayKey: false,
      });
      final native = _FakeNativePreferences(initialized: false);
      final store = _nativeStore(native);

      final migrated = await store.load();
      expect(migrated.preferences.serviceEnabled, isTrue);
      expect(migrated.preferences.gatewayEnabled, isFalse);
      expect(migrated.source, 'native_canonical_migrated_from_flutter');
      expect(native.initialized, isTrue);

      // Once initialized, Android is canonical and stale Flutter mirror values
      // cannot overwrite a Service-visible preference.
      final prefs = await SharedPreferences.getInstance();
      await prefs.setBool(LaunchAutoStartPreferenceStore.gatewayKey, true);
      final reloaded = await _nativeStore(native).load();
      expect(reloaded.preferences.gatewayEnabled, isFalse);
      expect(prefs.getBool(LaunchAutoStartPreferenceStore.gatewayKey), isFalse);
    },
  );

  test(
    'Flutter update and Service restart observe identical native values',
    () async {
      final native = _FakeNativePreferences(initialized: true);
      final store = _nativeStore(native);

      final updated = await store.update(
        serviceEnabled: true,
        gatewayEnabled: true,
      );
      final serviceRestartRead = await native.read();

      expect(
        updated.preferences.serviceEnabled,
        serviceRestartRead.serviceEnabled,
      );
      expect(
        updated.preferences.gatewayEnabled,
        serviceRestartRead.gatewayEnabled,
      );
      expect(serviceRestartRead.gatewayEnabled, isTrue);
    },
  );

  test(
    'toggle OFF to ON is verified by canonical readback and logged',
    () async {
      final native = _FakeNativePreferences(initialized: true)
        ..serviceEnabled = false;
      final events = <({String event, Map<String, Object?> metadata})>[];

      final updated = await _nativeStore(native).update(
        serviceEnabled: true,
        operationId: 'preference-service-1',
        logger: (event, metadata) {
          events.add((event: event, metadata: metadata));
        },
      );

      expect(updated.preferences.serviceEnabled, isTrue);
      expect((await native.read()).serviceEnabled, isTrue);
      expect(events.map((entry) => entry.event), <String>[
        'autostart.preference.change.requested',
        'autostart.preference.change.persisted',
        'autostart.preference.readback',
      ]);
      expect(events.last.metadata, containsPair('persisted_value', true));
      expect(events.last.metadata, containsPair('result', 'verified'));
      expect(
        events.last.metadata,
        containsPair('preference_source', 'android_native_canonical'),
      );
    },
  );

  test('toggle ON to OFF is verified by canonical readback', () async {
    final native = _FakeNativePreferences(initialized: true);

    final updated = await _nativeStore(
      native,
    ).update(gatewayEnabled: false, operationId: 'preference-gateway-1');

    expect(updated.preferences.gatewayEnabled, isFalse);
    expect((await native.read()).gatewayEnabled, isFalse);
  });

  test(
    'failed canonical readback never updates Flutter mirror optimistically',
    () async {
      SharedPreferences.setMockInitialValues({
        LaunchAutoStartPreferenceStore.serviceKey: false,
      });
      final native = _FakeNativePreferences(initialized: true)
        ..serviceEnabled = false
        ..ignoreWrites = true;
      final events = <String>[];

      await expectLater(
        _nativeStore(native).update(
          serviceEnabled: true,
          operationId: 'preference-service-failed',
          logger: (event, _) => events.add(event),
        ),
        throwsStateError,
      );

      expect((await native.read()).serviceEnabled, isFalse);
      final flutter = await SharedPreferences.getInstance();
      expect(
        flutter.getBool(LaunchAutoStartPreferenceStore.serviceKey),
        isFalse,
      );
      expect(events, contains('autostart.preference.change.failed'));
    },
  );

  test(
    'rapid independent toggles are serialized without losing a value',
    () async {
      final native = _FakeNativePreferences(initialized: true);
      final store = _nativeStore(native);

      await Future.wait([
        store.update(serviceEnabled: false),
        store.update(gatewayEnabled: false),
        store.update(serviceEnabled: true),
        store.update(gatewayEnabled: true),
      ]);

      final snapshot = await store.load();
      expect(snapshot.preferences.serviceEnabled, isTrue);
      expect(snapshot.preferences.gatewayEnabled, isTrue);
    },
  );
}

LaunchAutoStartPreferenceStore _flutterStore() {
  return LaunchAutoStartPreferenceStore(
    useNativeStore: false,
    readNative: () => throw StateError('native read must not be used'),
    writeNative: ({serviceEnabled, gatewayEnabled}) =>
        throw StateError('native write must not be used'),
  );
}

LaunchAutoStartPreferenceStore _nativeStore(_FakeNativePreferences native) {
  return LaunchAutoStartPreferenceStore(
    useNativeStore: true,
    readNative: native.read,
    writeNative: native.update,
  );
}

class _FakeNativePreferences {
  _FakeNativePreferences({required this.initialized});

  bool serviceEnabled = true;
  bool gatewayEnabled = true;
  bool initialized;
  bool ignoreWrites = false;

  Future<NativeLaunchAutoStartPreferences> read() async => _snapshot();

  Future<NativeLaunchAutoStartPreferences> update({
    bool? serviceEnabled,
    bool? gatewayEnabled,
  }) async {
    await Future<void>.delayed(Duration.zero);
    if (!ignoreWrites) {
      this.serviceEnabled = serviceEnabled ?? this.serviceEnabled;
      this.gatewayEnabled = gatewayEnabled ?? this.gatewayEnabled;
    }
    initialized = true;
    return _snapshot();
  }

  NativeLaunchAutoStartPreferences _snapshot() {
    return NativeLaunchAutoStartPreferences(
      serviceEnabled: serviceEnabled,
      gatewayEnabled: gatewayEnabled,
      initialized: initialized,
      source: 'android_native_canonical',
    );
  }
}
