import 'dart:async';

import 'package:shared_preferences/shared_preferences.dart';

import 'autostart_coordinator.dart';

class NativeLaunchAutoStartPreferences {
  const NativeLaunchAutoStartPreferences({
    required this.serviceEnabled,
    required this.gatewayEnabled,
    required this.initialized,
    required this.source,
  });

  final bool serviceEnabled;
  final bool gatewayEnabled;
  final bool initialized;
  final String source;
}

typedef NativeLaunchAutoStartReader =
    Future<NativeLaunchAutoStartPreferences> Function();
typedef NativeLaunchAutoStartWriter =
    Future<NativeLaunchAutoStartPreferences> Function({
      bool? serviceEnabled,
      bool? gatewayEnabled,
    });

class LaunchAutoStartPreferenceSnapshot {
  const LaunchAutoStartPreferenceSnapshot({
    required this.preferences,
    required this.source,
  });

  final AutoStartPreferences preferences;
  final String source;
}

/// Owns the migration and mirroring contract for launch auto-start settings.
///
/// Android's app-private native preference file is canonical because the
/// foreground Service must be able to read it when Flutter is not running.
/// Flutter SharedPreferences remains a mirror for upgrade compatibility and
/// non-Android platforms.
class LaunchAutoStartPreferenceStore {
  LaunchAutoStartPreferenceStore({
    required this.useNativeStore,
    required NativeLaunchAutoStartReader readNative,
    required NativeLaunchAutoStartWriter writeNative,
  }) : _readNative = readNative,
       _writeNative = writeNative;

  static const serviceKey = 'service_launch_autostart';
  static const gatewayKey = 'gateway_launch_autostart';

  final bool useNativeStore;
  final NativeLaunchAutoStartReader _readNative;
  final NativeLaunchAutoStartWriter _writeNative;

  Future<void> _serializedWrite = Future<void>.value();

  Future<LaunchAutoStartPreferenceSnapshot> load() async {
    await _serializedWrite;
    final prefs = await SharedPreferences.getInstance();
    if (!useNativeStore) {
      return LaunchAutoStartPreferenceSnapshot(
        preferences: _readFlutterPreferences(prefs),
        source: 'flutter_shared_preferences',
      );
    }

    var native = await _readNative();
    if (!native.initialized) {
      final hasFlutterValue =
          prefs.containsKey(serviceKey) || prefs.containsKey(gatewayKey);
      final migration = _readFlutterPreferences(prefs);
      native = await _writeNative(
        serviceEnabled: migration.serviceEnabled,
        gatewayEnabled: migration.gatewayEnabled,
      );
      await _writeFlutterMirror(prefs, native);
      return LaunchAutoStartPreferenceSnapshot(
        preferences: _fromNative(native),
        source: hasFlutterValue
            ? 'native_canonical_migrated_from_flutter'
            : 'native_canonical_defaults',
      );
    }

    await _writeFlutterMirror(prefs, native);
    return LaunchAutoStartPreferenceSnapshot(
      preferences: _fromNative(native),
      source: native.source.isEmpty ? 'native_canonical' : native.source,
    );
  }

  Future<LaunchAutoStartPreferenceSnapshot> update({
    bool? serviceEnabled,
    bool? gatewayEnabled,
  }) {
    final result = Completer<LaunchAutoStartPreferenceSnapshot>();
    _serializedWrite = _serializedWrite.then((_) async {
      try {
        result.complete(
          await _updateInternal(
            serviceEnabled: serviceEnabled,
            gatewayEnabled: gatewayEnabled,
          ),
        );
      } catch (error, stackTrace) {
        result.completeError(error, stackTrace);
      }
    });
    return result.future;
  }

  Future<LaunchAutoStartPreferenceSnapshot> _updateInternal({
    bool? serviceEnabled,
    bool? gatewayEnabled,
  }) async {
    final prefs = await SharedPreferences.getInstance();
    if (!useNativeStore) {
      final current = _readFlutterPreferences(prefs);
      final updated = AutoStartPreferences(
        serviceEnabled: serviceEnabled ?? current.serviceEnabled,
        gatewayEnabled: gatewayEnabled ?? current.gatewayEnabled,
      );
      await prefs.setBool(serviceKey, updated.serviceEnabled);
      await prefs.setBool(gatewayKey, updated.gatewayEnabled);
      return LaunchAutoStartPreferenceSnapshot(
        preferences: updated,
        source: 'flutter_shared_preferences',
      );
    }

    final native = await _writeNative(
      serviceEnabled: serviceEnabled,
      gatewayEnabled: gatewayEnabled,
    );
    await _writeFlutterMirror(prefs, native);
    return LaunchAutoStartPreferenceSnapshot(
      preferences: _fromNative(native),
      source: native.source.isEmpty ? 'native_canonical' : native.source,
    );
  }

  AutoStartPreferences _readFlutterPreferences(SharedPreferences prefs) {
    return AutoStartPreferences(
      serviceEnabled:
          prefs.getBool(serviceKey) ??
          AutoStartPreferences.defaults.serviceEnabled,
      gatewayEnabled:
          prefs.getBool(gatewayKey) ??
          AutoStartPreferences.defaults.gatewayEnabled,
    );
  }

  AutoStartPreferences _fromNative(NativeLaunchAutoStartPreferences native) {
    return AutoStartPreferences(
      serviceEnabled: native.serviceEnabled,
      gatewayEnabled: native.gatewayEnabled,
    );
  }

  Future<void> _writeFlutterMirror(
    SharedPreferences prefs,
    NativeLaunchAutoStartPreferences native,
  ) async {
    await prefs.setBool(serviceKey, native.serviceEnabled);
    await prefs.setBool(gatewayKey, native.gatewayEnabled);
  }
}
