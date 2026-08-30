import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/ui/config_page.dart';

/// The Settings card must show the persisted preference and the live runtime
/// as two separate facts. Conflating them is what made a stopped service look
/// enabled and a running one look failed on the device.
void main() {
  Future<void> pump(
    WidgetTester tester, {
    required bool serviceEnabled,
    required bool gatewayEnabled,
    required ServiceStatus serviceStatus,
    ValueChanged<bool>? onServiceChanged,
    ValueChanged<bool>? onGatewayChanged,
  }) => tester.pumpWidget(
    MaterialApp(
      home: Scaffold(
        body: AutoStartSettingsCard(
          serviceEnabled: serviceEnabled,
          gatewayEnabled: gatewayEnabled,
          serviceStatus: serviceStatus,
          onServiceChanged: onServiceChanged ?? (_) {},
          onGatewayChanged: onGatewayChanged ?? (_) {},
        ),
      ),
    ),
  );

  testWidgets('shows preference and runtime as separate lines', (tester) async {
    await pump(
      tester,
      serviceEnabled: true,
      gatewayEnabled: true,
      serviceStatus: ServiceStatus.stopped,
    );

    // Preference ON while the runtime is Stopped is a legitimate state, and the
    // card has to say both.
    expect(
      find.textContaining('Auto-start preference: ON'),
      findsNWidgets(2),
    );
    expect(find.textContaining('Runtime: Stopped'), findsOneWidget);
  });

  testWidgets('reflects a manual start without the preference changing', (
    tester,
  ) async {
    await pump(
      tester,
      serviceEnabled: false,
      gatewayEnabled: false,
      serviceStatus: ServiceStatus.running,
    );

    expect(find.textContaining('Auto-start preference: OFF'), findsNWidgets(2));
    expect(find.textContaining('Runtime: Running'), findsOneWidget);
  });

  testWidgets('reports a starting service', (tester) async {
    await pump(
      tester,
      serviceEnabled: true,
      gatewayEnabled: true,
      serviceStatus: ServiceStatus.starting,
    );

    expect(find.textContaining('Runtime: Starting'), findsOneWidget);
  });

  testWidgets('toggling service auto-start reports the new value', (
    tester,
  ) async {
    final changes = <bool>[];
    await pump(
      tester,
      serviceEnabled: true,
      gatewayEnabled: true,
      serviceStatus: ServiceStatus.stopped,
      onServiceChanged: changes.add,
    );

    await tester.tap(
      find.text('Start PocketClaw service automatically'),
    );
    await tester.pumpAndSettle();

    expect(changes, <bool>[false]);
  });

  testWidgets('toggling gateway auto-start reports the new value', (
    tester,
  ) async {
    final changes = <bool>[];
    await pump(
      tester,
      serviceEnabled: true,
      gatewayEnabled: false,
      serviceStatus: ServiceStatus.running,
      onGatewayChanged: changes.add,
    );

    await tester.tap(find.text('Start Gateway automatically'));
    await tester.pumpAndSettle();

    expect(changes, <bool>[true]);
  });

  testWidgets('gateway row does not claim to control gateway runtime', (
    tester,
  ) async {
    await pump(
      tester,
      serviceEnabled: true,
      gatewayEnabled: true,
      serviceStatus: ServiceStatus.running,
    );

    expect(
      find.textContaining('Gateway runtime is managed in the Dashboard'),
      findsOneWidget,
    );
  });
}
