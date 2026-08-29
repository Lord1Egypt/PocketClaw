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
    String? operationId,
    AutoStartLifecycleLogger? logger,
  }) {
    final result = Completer<LaunchAutoStartPreferenceSnapshot>();
    _serializedWrite = _serializedWrite.then((_) async {
      try {
        result.complete(
          await _updateInternal(
            serviceEnabled: serviceEnabled,
            gatewayEnabled: gatewayEnabled,
            operationId:
                operationId ??
                'preference-${DateTime.now().toUtc().microsecondsSinceEpoch}',
            logger: logger,
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
    required String operationId,
    AutoStartLifecycleLogger? logger,
  }) async {
    final prefs = await SharedPreferences.getInstance();
    if (!useNativeStore) {
      final current = _readFlutterPreferences(prefs);
      return _writeAndVerify(
        previous: current,
        source: 'flutter_shared_preferences',
        serviceEnabled: serviceEnabled,
        gatewayEnabled: gatewayEnabled,
        operationId: operationId,
        logger: logger,
        write: () async {
          final updated = AutoStartPreferences(
            serviceEnabled: serviceEnabled ?? current.serviceEnabled,
            gatewayEnabled: gatewayEnabled ?? current.gatewayEnabled,
          );
          final servicePersisted = await prefs.setBool(
            serviceKey,
            updated.serviceEnabled,
          );
          final gatewayPersisted = await prefs.setBool(
            gatewayKey,
            updated.gatewayEnabled,
          );
          if (!servicePersisted || !gatewayPersisted) {
            throw StateError('Could not persist launch auto-start preferences');
          }
          return _readFlutterPreferences(prefs);
        },
        readBack: () async => _readFlutterPreferences(prefs),
      );
    }

    final previousNative = await _readNative();
    final previous = _fromNative(previousNative);
    return _writeAndVerify(
      previous: previous,
      source: previousNative.source.isEmpty
          ? 'android_native_canonical'
          : previousNative.source,
      serviceEnabled: serviceEnabled,
      gatewayEnabled: gatewayEnabled,
      operationId: operationId,
      logger: logger,
      write: () async {
        final written = await _writeNative(
          serviceEnabled: serviceEnabled,
          gatewayEnabled: gatewayEnabled,
        );
        return _fromNative(written);
      },
      readBack: () async {
        final canonical = await _readNative();
        return _fromNative(canonical);
      },
      mirror: (verified) async {
        await _writeFlutterMirror(
          prefs,
          NativeLaunchAutoStartPreferences(
            serviceEnabled: verified.serviceEnabled,
            gatewayEnabled: verified.gatewayEnabled,
            initialized: true,
            source: 'android_native_canonical',
          ),
        );
      },
    );
  }

  Future<LaunchAutoStartPreferenceSnapshot> _writeAndVerify({
    required AutoStartPreferences previous,
    required String source,
    required bool? serviceEnabled,
    required bool? gatewayEnabled,
    required String operationId,
    required AutoStartLifecycleLogger? logger,
    required Future<AutoStartPreferences> Function() write,
    required Future<AutoStartPreferences> Function() readBack,
    Future<void> Function(AutoStartPreferences verified)? mirror,
  }) async {
    final mutations = <({String name, bool previous, bool requested})>[
      if (serviceEnabled != null)
        (
          name: 'service_autostart',
          previous: previous.serviceEnabled,
          requested: serviceEnabled,
        ),
      if (gatewayEnabled != null)
        (
          name: 'gateway_autostart',
          previous: previous.gatewayEnabled,
          requested: gatewayEnabled,
        ),
    ];
    for (final mutation in mutations) {
      _logPreferenceMutation(
        logger,
        'autostart.preference.change.requested',
        operationId: operationId,
        source: source,
        mutation: mutation,
        persistedValue: mutation.previous,
        result: 'requested',
      );
    }

    AutoStartPreferences? written;
    AutoStartPreferences? verified;
    try {
      written = await write();
      for (final mutation in mutations) {
        final persistedValue = _valueFor(written, mutation.name);
        _logPreferenceMutation(
          logger,
          'autostart.preference.change.persisted',
          operationId: operationId,
          source: source,
          mutation: mutation,
          persistedValue: persistedValue,
          result: persistedValue == mutation.requested
              ? 'persisted'
              : 'mismatch',
        );
      }

      // The write response is not treated as proof. Read the canonical store
      // again and drive both the Flutter mirror and UI from that exact value.
      verified = await readBack();
      var matchesRequested = true;
      for (final mutation in mutations) {
        final persistedValue = _valueFor(verified, mutation.name);
        final matches = persistedValue == mutation.requested;
        matchesRequested = matchesRequested && matches;
        _logPreferenceMutation(
          logger,
          'autostart.preference.readback',
          operationId: operationId,
          source: source,
          mutation: mutation,
          persistedValue: persistedValue,
          result: matches ? 'verified' : 'mismatch',
        );
      }
      if (!matchesRequested) {
        throw StateError('Canonical launch auto-start readback did not match');
      }
      await mirror?.call(verified);
      return LaunchAutoStartPreferenceSnapshot(
        preferences: verified,
        source: source,
      );
    } catch (_) {
      for (final mutation in mutations) {
        _logPreferenceMutation(
          logger,
          'autostart.preference.change.failed',
          operationId: operationId,
          source: source,
          mutation: mutation,
          persistedValue: verified == null
              ? (written == null
                    ? mutation.previous
                    : _valueFor(written, mutation.name))
              : _valueFor(verified, mutation.name),
          result: 'failed',
        );
      }
      rethrow;
    }
  }

  bool _valueFor(AutoStartPreferences preferences, String name) =>
      name == 'service_autostart'
      ? preferences.serviceEnabled
      : preferences.gatewayEnabled;

  void _logPreferenceMutation(
    AutoStartLifecycleLogger? logger,
    String event, {
    required String operationId,
    required String source,
    required ({String name, bool previous, bool requested}) mutation,
    required bool persistedValue,
    required String result,
  }) {
    logger?.call(event, <String, Object?>{
      'preference': mutation.name,
      'previous_value': mutation.previous,
      'requested_value': mutation.requested,
      'persisted_value': persistedValue,
      'preference_source': source,
      'operation_id': operationId,
      'result': result,
    });
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
