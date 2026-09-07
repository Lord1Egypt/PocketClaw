import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/core/app_theme.dart';
import 'package:pocketclaw/src/core/status_snapshot.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/status_sections.dart';

Widget _host(Widget child, {Locale locale = const Locale('en')}) {
  return MaterialApp(
    locale: locale,
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: AppLocalizations.supportedLocales,
    theme: ApertureTheme.light(AppThemeMode.carbon),
    home: Scaffold(body: SingleChildScrollView(child: child)),
  );
}

StatusSnapshot _snapshot(Map<String, Object?> overrides) {
  final base = <String, Object?>{
    'activity': {
      'active_turns': 1,
      'active_subagents': 0,
      'waiting': 0,
      'completed': 12,
      'failed': 1,
      'cancelled': 2,
      'tool_calls': 48,
      'tool_calls_failed': 2,
      'last_activity_unix': 0,
    },
    'model': {
      'active_model': 'mimo-v2.5',
      'configured_model': 'mimo-v2.5',
      'provider': 'openai',
      'fallback_count': 3,
    },
    'channels': [
      {
        'name': 'telegram',
        'configured': true,
        'started': true,
        'running': true,
      },
    ],
    'resources': {'memory_rss_bytes': 88080384, 'cpu_seconds': 252.5},
  };
  base.addAll(overrides);
  return StatusSnapshot.tryParse(jsonEncode(base))!;
}

void main() {
  testWidgets('renders with the gateway stopped and no detail', (tester) async {
    await tester.pumpWidget(
      _host(
        const StatusSections(
          gatewayRunning: false,
          uptime: '',
          appVersion: '',
          coreVersion: '',
          snapshot: null,
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(find.text('Stopped'), findsOneWidget);
    // Detail is stated as unavailable rather than rendered as zeroes.
    expect(find.text('Detailed status is unavailable'), findsOneWidget);
    expect(find.text('Completed'), findsNothing);
    // Unknown values read as an em dash, never as an empty row or a 0.
    expect(find.text('—'), findsWidgets);
  });

  testWidgets('renders a populated snapshot', (tester) async {
    await tester.pumpWidget(
      _host(
        StatusSections(
          gatewayRunning: true,
          uptime: '1h24m0s',
          appVersion: '0.2.0',
          coreVersion: '0.2.0',
          snapshot: _snapshot(const {}),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(find.text('Running'), findsNWidgets(2)); // gateway + telegram
    expect(find.text('Since Gateway start'), findsOneWidget);
    expect(find.text('12'), findsOneWidget);
    expect(find.text('48'), findsOneWidget);
    expect(find.text('84 MB'), findsOneWidget);
    expect(find.text('4m 13s'), findsOneWidget);
    expect(find.text('telegram'), findsOneWidget);
  });

  testWidgets('renders zeroed counters without collapsing', (tester) async {
    await tester.pumpWidget(
      _host(
        StatusSections(
          gatewayRunning: true,
          uptime: '0s',
          appVersion: '0.2.0',
          coreVersion: '0.2.0',
          snapshot: _snapshot(const {
            'activity': <String, Object?>{},
            'channels': <Object?>[],
            'resources': <String, Object?>{},
          }),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(find.text('No channels configured'), findsOneWidget);
    expect(find.text('Completed'), findsOneWidget);
    // Zero resources are unavailable, not "0 MB".
    expect(find.text('0 MB'), findsNothing);
  });

  testWidgets('shows the configured default only when it diverges', (
    tester,
  ) async {
    await tester.pumpWidget(
      _host(
        StatusSections(
          gatewayRunning: true,
          uptime: '1m',
          appVersion: '0.2.0',
          coreVersion: '0.2.0',
          snapshot: _snapshot(const {}),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Configured default'), findsNothing);

    await tester.pumpWidget(
      _host(
        StatusSections(
          gatewayRunning: true,
          uptime: '1m',
          appVersion: '0.2.0',
          coreVersion: '0.2.0',
          snapshot: _snapshot(const {
            'model': {
              'active_model': 'mimo-v2.5',
              'configured_model': 'gpt-5.6-sol',
              'provider': 'openai',
              'fallback_count': 3,
            },
          }),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Configured default'), findsOneWidget);
    expect(find.text('gpt-5.6-sol'), findsOneWidget);
    expect(find.text('mimo-v2.5'), findsOneWidget);
  });

  testWidgets('a channel that failed to start is never shown as running', (
    tester,
  ) async {
    await tester.pumpWidget(
      _host(
        StatusSections(
          gatewayRunning: true,
          uptime: '1m',
          appVersion: '0.2.0',
          coreVersion: '0.2.0',
          snapshot: _snapshot(const {
            'channels': [
              {
                'name': 'discord',
                'configured': true,
                'started': false,
                'running': false,
              },
              // A latched running flag with no worker must not read as
              // Running either.
              {
                'name': 'matrix',
                'configured': true,
                'started': false,
                'running': true,
              },
            ],
          }),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Failed to start'), findsNWidgets(2));
    // Only the gateway row says Running.
    expect(find.text('Running'), findsOneWidget);
  });

  testWidgets('lays out under an RTL locale without overflowing', (
    tester,
  ) async {
    await tester.pumpWidget(
      _host(
        StatusSections(
          gatewayRunning: true,
          uptime: '1h24m0s',
          appVersion: '0.2.0',
          coreVersion: '0.2.0',
          snapshot: _snapshot(const {}),
        ),
        locale: const Locale('ar'),
      ),
    );
    await tester.pumpAndSettle();

    expect(tester.takeException(), isNull);
    expect(
      Directionality.of(tester.element(find.byType(StatusSections))),
      TextDirection.rtl,
    );
    expect(find.text('النشاط'), findsOneWidget);
    expect(find.text('منذ بدء البوابة'), findsOneWidget);
  });
}
