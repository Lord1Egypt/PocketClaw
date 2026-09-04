import 'package:package_info_plus/package_info_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'whats_new_release.dart';

/// Remembers which release the user has already read the notes for.
abstract class WhatsNewSeenStore {
  Future<String?> readLastSeenVersion();
  Future<void> writeLastSeenVersion(String version);
}

class SharedPreferencesWhatsNewSeenStore implements WhatsNewSeenStore {
  const SharedPreferencesWhatsNewSeenStore();

  static const String preferenceKey = 'pocketclaw.whats_new.last_seen_version';

  @override
  Future<String?> readLastSeenVersion() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(preferenceKey);
  }

  @override
  Future<void> writeLastSeenVersion(String version) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(preferenceKey, version);
  }
}

/// Reads the release identity from the platform package.
///
/// Deliberately `version` only. `buildNumber` is the Android versionCode, and
/// that is bumped for internal candidates that carry nothing new to announce.
Future<String> readWhatsNewReleaseVersion() async {
  try {
    final info = await PackageInfo.fromPlatform();
    final version = info.version.trim();
    if (version.isNotEmpty) return version;
  } catch (_) {
    // No platform channel (tests, unusual hosts): fall through.
  }
  return currentWhatsNewRelease.version;
}
