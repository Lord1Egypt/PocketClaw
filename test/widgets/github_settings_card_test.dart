import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/ui/github_settings_card.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.lord1egypt.pocketclaw/picoclaw');
  const canary = 'ghp_canary_widget_must_never_render_this';

  final calls = <MethodCall>[];

  void mockNative(Object? Function(MethodCall call) respond) {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          calls.add(call);
          return respond(call);
        });
  }

  setUp(calls.clear);

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  Future<void> pumpCard(WidgetTester tester) async {
    await tester.pumpWidget(
      const MaterialApp(home: Scaffold(body: GitHubSettingsCard())),
    );
    await tester.pumpAndSettle();
  }

  testWidgets('offers to connect when no credential is stored', (tester) async {
    mockNative((_) => <String, Object?>{'connected': false, 'login': null});

    await pumpCard(tester);

    expect(find.byKey(const Key('github-connect-button')), findsOneWidget);
    expect(find.byKey(const Key('github-disconnect-button')), findsNothing);
    expect(find.text('Not connected'), findsOneWidget);
  });

  testWidgets('names the connected account and offers no reveal', (
    tester,
  ) async {
    mockNative((_) => <String, Object?>{'connected': true, 'login': 'octocat'});

    await pumpCard(tester);

    expect(find.text('Connected as octocat'), findsOneWidget);
    expect(find.byKey(const Key('github-disconnect-button')), findsOneWidget);
    expect(find.byKey(const Key('github-test-button')), findsOneWidget);

    // There is no control that could show a stored credential, and nothing on
    // screen is a text field the value could be recovered from.
    expect(find.textContaining('Reveal'), findsNothing);
    expect(find.textContaining('Show token'), findsNothing);
    expect(find.byType(TextField), findsNothing);
  });

  testWidgets('sends the token to the platform and never renders it', (
    tester,
  ) async {
    var connected = false;
    mockNative((call) {
      switch (call.method) {
        case 'connectGitHub':
          connected = true;
          return <String, Object?>{'connected': true, 'login': 'octocat'};
        default:
          return <String, Object?>{
            'connected': connected,
            'login': connected ? 'octocat' : null,
          };
      }
    });

    await pumpCard(tester);
    await tester.tap(find.byKey(const Key('github-connect-button')));
    await tester.pumpAndSettle();

    await tester.enterText(find.byKey(const Key('github-token-field')), canary);
    await tester.tap(find.byKey(const Key('github-token-save')));
    await tester.pumpAndSettle();

    final connectCall = calls.firstWhere((c) => c.method == 'connectGitHub');
    expect((connectCall.arguments as Map)['token'], canary);

    expect(find.text('Connected as octocat'), findsOneWidget);
    // The dialog is gone and the value is nowhere in the rendered tree.
    expect(find.byKey(const Key('github-token-field')), findsNothing);
    expect(find.textContaining(canary), findsNothing);
  });

  testWidgets('reports a rejected token without echoing it', (tester) async {
    mockNative((call) {
      if (call.method == 'connectGitHub') {
        throw PlatformException(
          code: 'GITHUB_CONNECT_FAILED',
          message: 'GitHub rejected this token: HTTP 401 [redacted]',
        );
      }
      return <String, Object?>{'connected': false, 'login': null};
    });

    await pumpCard(tester);
    await tester.tap(find.byKey(const Key('github-connect-button')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('github-token-field')), canary);
    await tester.tap(find.byKey(const Key('github-token-save')));
    await tester.pumpAndSettle();

    expect(find.textContaining('HTTP 401'), findsOneWidget);
    expect(find.textContaining(canary), findsNothing);
    expect(find.byKey(const Key('github-connect-button')), findsOneWidget);
  });

  testWidgets('disconnect asks the platform to remove the credential', (
    tester,
  ) async {
    var connected = true;
    mockNative((call) {
      if (call.method == 'disconnectGitHub') {
        connected = false;
        return true;
      }
      return <String, Object?>{
        'connected': connected,
        'login': connected ? 'octocat' : null,
      };
    });

    await pumpCard(tester);
    await tester.tap(find.byKey(const Key('github-disconnect-button')));
    await tester.pumpAndSettle();

    expect(calls.map((c) => c.method), contains('disconnectGitHub'));
    expect(find.text('Not connected'), findsOneWidget);
    expect(find.byKey(const Key('github-connect-button')), findsOneWidget);
  });
}
