import 'core_service_adapter.dart';
import 'android_core_service_adapter.dart';

class CoreServiceAdapterFactory {
  /// PocketClaw is Android-only: the repository carries no other Flutter
  /// platform, and Core runs as the Android service's child process.
  static CoreServiceAdapter create() => AndroidCoreServiceAdapter();
}
