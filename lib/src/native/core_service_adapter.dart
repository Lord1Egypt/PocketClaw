import 'dart:async';

abstract class CoreServiceAdapter {
  Future<bool> startService({int? port, String? args});
  Future<bool> stopService();
  Future<Map<String, dynamic>> getServiceStatus();
  /// Polls the gateway's health endpoint.
  ///
  /// When [detail] is true the richer Status snapshot is requested as well.
  /// That request is authenticated by the host with the gateway's own
  /// credential, which never crosses into Dart, and it is only asked for while
  /// the Status screen is visible.
  Future<Map<String, dynamic>> checkHealth({bool detail = false});
  Future<bool> setAutoStart(bool enabled);
  Future<bool> getAutoStart();
  Future<String> getCoreVersion();
  void setConfiguredPath(String? path);
  String? getLastErrorCode();

  /// Install a log handler callback which the adapter should call with
  /// each new log line (or combined message). Pass `null` to clear.
  void setLogHandler(void Function(String)? handler);

  /// Validate that the binary (optionally at [path]) is present and usable.
  /// Returns true if valid; adapters should set an internal last error code
  /// accessible via `getLastErrorCode()` on failure.
  Future<bool> validateBinary([String? path]);

  /// Read the current workspace path from the platform config.
  /// Returns an empty string if not set or not supported.
  Future<String> getWorkspacePath();

  /// Write the workspace path into the platform config.
  /// Returns true on success.
  Future<bool> setWorkspacePath(String path);
}
