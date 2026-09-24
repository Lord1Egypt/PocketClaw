import 'dart:async';

abstract class CoreServiceAdapter {
  Future<bool> startService({int? port, String? args});
  Future<bool> stopService();

  /// Stops and starts Core as one operation.
  ///
  /// PC-DEF-030. A caller that issues stopService() then startService() is
  /// expressing a restart the platform cannot honour as two requests: on
  /// Android the stop ends in an unconditional stopSelf(), which tears down the
  /// service the start has already asked for, so Core is left stopped and the
  /// user has to start it by hand. The platform is told "restart" instead.
  Future<bool> restartService({int? port, String? args});
  Future<Map<String, dynamic>> getServiceStatus();

  /// Polls the gateway's health endpoint.
  ///
  /// When [detail] is true the richer Status snapshot is requested as well.
  /// That request is authenticated by the host with the gateway's own
  /// credential, which never crosses into Dart, and it is only asked for while
  /// the Status screen is visible.
  Future<Map<String, dynamic>> checkHealth({bool detail = false});

  /// The Core runtime version, or null when it could not be read.
  ///
  /// Null rather than a sentinel string: reading it means running the Core
  /// binary, which can fail transiently, and a caller that cannot tell a
  /// failure from an answer caches the failure and shows it as the version
  /// (PC-DEF-063).
  Future<String?> getCoreVersion();
  String? getLastErrorCode();

  /// Read the current workspace path from the platform config.
  /// Returns an empty string if not set or not supported.
  Future<String> getWorkspacePath();
}
