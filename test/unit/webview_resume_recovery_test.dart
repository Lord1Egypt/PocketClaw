import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/ui/webview/pocketclaw_host_bridge.dart';

void main() {
  group('resume liveness probe', () {
    // The Android host cannot be told when a backgrounded WebView's renderer is
    // killed: webview_flutter_android exposes no onRenderProcessGone callback.
    // Everything here defends the substitute signal.

    test('only an explicit alive result counts as a healthy page', () {
      expect(PocketClawHostBridge.isAliveResult('alive'), isTrue);
      expect(PocketClawHostBridge.isAliveResult('"alive"'), isTrue);
      expect(PocketClawHostBridge.isAliveResult(' alive '), isTrue);
    });

    test('a dead or unrendered page is never mistaken for a healthy one', () {
      // null is what a renderer that cannot run script leaves behind.
      expect(PocketClawHostBridge.isAliveResult(null), isFalse);
      expect(PocketClawHostBridge.isAliveResult('not-ready'), isFalse);
      expect(PocketClawHostBridge.isAliveResult('empty'), isFalse);
      expect(PocketClawHostBridge.isAliveResult('error'), isFalse);
      expect(PocketClawHostBridge.isAliveResult(''), isFalse);
      expect(PocketClawHostBridge.isAliveResult(false), isFalse);
    });

    test('the probe asks for the readiness flag and for rendered content', () {
      final script = PocketClawHostBridge.livenessProbeScript;
      // Both halves matter: a bundle can parse without rendering, and a React
      // tree can unmount and leave the flag behind.
      expect(script.contains('__pocketclawReady'), isTrue);
      expect(script.contains('childElementCount'), isTrue);
      // It must never throw out of the page; a throw is indistinguishable from
      // a dead renderer and would misreport a live page.
      expect(script.contains('catch'), isTrue);
    });
  });

  group('route preservation', () {
    // Recovery must return the user where they were. Reloading the console's
    // home page would silently discard their navigation.

    test('an in-app route is accepted', () {
      expect(PocketClawHostBridge.routeFromResult('/models'), '/models');
      expect(
        PocketClawHostBridge.routeFromResult('"/logs?tail=1"'),
        '/logs?tail=1',
      );
    });

    test('anything that is not an in-app route is rejected', () {
      // A dead renderer returns nothing; an absolute URL is not this console's
      // route and must not be navigated to on its behalf.
      expect(PocketClawHostBridge.routeFromResult(null), isNull);
      expect(PocketClawHostBridge.routeFromResult(''), isNull);
      expect(
        PocketClawHostBridge.routeFromResult('https://example.com/'),
        isNull,
      );
      expect(PocketClawHostBridge.routeFromResult('models'), isNull);
    });

    test('the route script cannot throw out of the page', () {
      expect(PocketClawHostBridge.currentRouteScript.contains('catch'), isTrue);
    });
  });
}
