import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/autostart_coordinator.dart';
import 'package:pocketclaw/src/core/launch_autostart_preferences.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/ui/config_page.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues({});
  });

  testWidgets('shows independent defaults and truthful Gateway dependency', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: AutoStartSettingsCard(
            serviceEnabled: false,
            gatewayEnabled: true,
            serviceStatus: ServiceStatus.stopped,
            gatewayStatus: AutoStartRuntimeState.stopped,
            onServiceChanged: (_) {},
            onGatewayChanged: (_) {},
          ),
        ),
      ),
    );

    expect(find.text('Start PocketClaw service automatically'), findsOneWidget);
    expect(find.text('Start Gateway automatically'), findsOneWidget);
    expect(
      find.textContaining(
        'Runtime: Requires a running PocketClaw service; this setting will not start it.',
      ),
      findsOneWidget,
    );

    final switches = tester.widgetList<Switch>(find.byType(Switch)).toList();
    expect(switches.map((item) => item.value), [false, true]);
  });

  testWidgets('runtime state is separate from enabled preference', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: AutoStartSettingsCard(
            serviceEnabled: true,
            gatewayEnabled: true,
            serviceStatus: ServiceStatus.failed,
            gatewayStatus: AutoStartRuntimeState.starting,
            onServiceChanged: (_) {},
            onGatewayChanged: (_) {},
          ),
        ),
      ),
    );

    expect(find.textContaining('Runtime: Service failed'), findsOneWidget);
    expect(find.textContaining('Runtime: Gateway starting'), findsOneWidget);
  });

  testWidgets('auto-start toggles persist across widget recreation', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(home: _PreferenceHarness(key: UniqueKey())),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Start PocketClaw service automatically'));
    await tester.pumpAndSettle();

    await tester.pumpWidget(
      MaterialApp(home: _PreferenceHarness(key: UniqueKey())),
    );
    await tester.pumpAndSettle();

    final switches = tester.widgetList<Switch>(find.byType(Switch)).toList();
    expect(switches.map((item) => item.value), [false, true]);
    expect(find.textContaining('Auto-start preference: OFF'), findsOneWidget);
    expect(find.textContaining('Runtime: Service running'), findsOneWidget);
  });
}

class _PreferenceHarness extends StatefulWidget {
  const _PreferenceHarness({super.key});

  @override
  State<_PreferenceHarness> createState() => _PreferenceHarnessState();
}

class _PreferenceHarnessState extends State<_PreferenceHarness> {
  late final LaunchAutoStartPreferenceStore store;
  LaunchAutoStartPreferenceSnapshot? snapshot;

  @override
  void initState() {
    super.initState();
    store = LaunchAutoStartPreferenceStore(
      useNativeStore: false,
      readNative: () => throw StateError('unused'),
      writeNative: ({serviceEnabled, gatewayEnabled}) =>
          throw StateError('unused'),
    );
    store.load().then((value) {
      if (mounted) setState(() => snapshot = value);
    });
  }

  Future<void> update({bool? service, bool? gateway}) async {
    final value = await store.update(
      serviceEnabled: service,
      gatewayEnabled: gateway,
    );
    if (mounted) setState(() => snapshot = value);
  }

  @override
  Widget build(BuildContext context) {
    final current = snapshot;
    if (current == null) return const CircularProgressIndicator();
    return Scaffold(
      body: AutoStartSettingsCard(
        serviceEnabled: current.preferences.serviceEnabled,
        gatewayEnabled: current.preferences.gatewayEnabled,
        serviceStatus: ServiceStatus.running,
        gatewayStatus: AutoStartRuntimeState.stopped,
        onServiceChanged: (value) => update(service: value),
        onGatewayChanged: (value) => update(gateway: value),
      ),
    );
  }
}
