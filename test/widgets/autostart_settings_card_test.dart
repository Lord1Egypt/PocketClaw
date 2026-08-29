import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/autostart_coordinator.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/ui/config_page.dart';

void main() {
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
      find.text(
        'Requires a running PocketClaw service; this setting will not start it.',
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

    expect(find.text('Runtime: Service failed'), findsOneWidget);
    expect(find.text('Runtime: Gateway starting'), findsOneWidget);
  });
}
