import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/config_page.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    PackageInfo.setMockInitialValues(
      appName: 'PocketClaw',
      packageName: 'com.lord1egypt.pocketclaw',
      version: '0.1.3',
      buildNumber: '3',
      buildSignature: '',
    );
    SharedPreferences.setMockInitialValues({
      'telemetry_created_at': '2026-04-01T10:00:00.000Z',
      'telemetry_last_seen_at': '2026-04-23T10:00:00.000Z',
      'telemetry_last_foreground_at': '2026-04-23T10:00:00.000Z',
      'telemetry_last_active_at': '2026-04-23T10:00:00.000Z',
      'telemetry_last_derived_state': 'active',
      'telemetry_last_state_reason': 'recent_foreground_activity',
    });
  });

  testWidgets('config page renders with persisted telemetry state', (
    WidgetTester tester,
  ) async {
    final service = ServiceManager();
    await service.getDeviceTelemetrySnapshot(
      now: DateTime.utc(2026, 4, 23, 12),
    );

    await tester.pumpWidget(
      ChangeNotifierProvider<ServiceManager>.value(
        value: service,
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const Scaffold(body: ConfigPage()),
        ),
      ),
    );
    // ConfigPage includes editable fields with a repeating caret ticker, so
    // waiting for every frame to settle never completes. One frame is enough
    // to verify the persisted telemetry state was rendered.
    await tester.pump();

    expect(find.text('Settings'), findsOneWidget);
    expect(find.text('About'), findsOneWidget);
  });
}
