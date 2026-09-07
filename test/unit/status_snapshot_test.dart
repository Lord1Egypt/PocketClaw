import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/status_snapshot.dart';

void main() {
  const fullPayload = '''
  {
    "activity": {
      "active_turns": 1, "active_subagents": 2, "waiting": 3,
      "completed": 12, "failed": 1, "cancelled": 2,
      "tool_calls": 48, "tool_calls_failed": 2,
      "last_activity_unix": 1757203200
    },
    "model": {
      "active_model": "mimo-v2.5",
      "configured_model": "gpt-5.6-sol",
      "provider": "openai",
      "fallback_count": 3
    },
    "channels": [
      {"name": "telegram", "configured": true, "started": true, "running": true},
      {"name": "discord", "configured": true, "started": false, "running": false}
    ],
    "resources": {"memory_rss_bytes": 88080384, "cpu_seconds": 252.5}
  }
  ''';

  test('parses a full snapshot', () {
    final snapshot = StatusSnapshot.tryParse(fullPayload)!;

    expect(snapshot.activity.activeTurns, 1);
    expect(snapshot.activity.activeSubagents, 2);
    expect(snapshot.activity.waiting, 3);
    expect(snapshot.activity.completed, 12);
    expect(snapshot.activity.failed, 1);
    expect(snapshot.activity.cancelled, 2);
    expect(snapshot.activity.toolCalls, 48);
    expect(snapshot.activity.toolCallsFailed, 2);
    expect(snapshot.activity.lastActivity, isNotNull);

    expect(snapshot.model.activeModel, 'mimo-v2.5');
    expect(snapshot.model.provider, 'openai');
    expect(snapshot.model.fallbackCount, 3);

    expect(snapshot.channels, hasLength(2));
    expect(snapshot.channels.first.name, 'telegram');
    expect(snapshot.channels.last.running, isFalse);
    expect(snapshot.channels.last.started, isFalse);

    expect(snapshot.resources.memoryRssBytes, 88080384);
    expect(snapshot.resources.cpuSeconds, 252.5);
  });

  test('reports divergence only when the two models differ', () {
    expect(StatusSnapshot.tryParse(fullPayload)!.model.hasDivergence, isTrue);

    final same = jsonEncode({
      'model': {
        'active_model': 'gpt-5.6-sol',
        'configured_model': 'gpt-5.6-sol',
        'provider': 'openai',
        'fallback_count': 1,
      },
    });
    expect(StatusSnapshot.tryParse(same)!.model.hasDivergence, isFalse);

    // A missing configured model is not a divergence to report either; it is
    // simply unknown.
    final partial = jsonEncode({
      'model': {'active_model': 'gpt-5.6-sol', 'configured_model': ''},
    });
    expect(StatusSnapshot.tryParse(partial)!.model.hasDivergence, isFalse);
  });

  test('treats a zero last-activity as never, not as the epoch', () {
    final snapshot = StatusSnapshot.tryParse(
      jsonEncode({
        'activity': {'completed': 0, 'last_activity_unix': 0},
      }),
    )!;
    expect(snapshot.activity.lastActivity, isNull);
  });

  test('returns null rather than an empty snapshot for unusable input', () {
    // Null means "unavailable" and the screen says so. An empty snapshot would
    // present zeroes as if they were measurements.
    expect(StatusSnapshot.tryParse(null), isNull);
    expect(StatusSnapshot.tryParse(''), isNull);
    expect(StatusSnapshot.tryParse('not json'), isNull);
    expect(StatusSnapshot.tryParse('[1,2,3]'), isNull);
  });

  test('tolerates missing and wrongly typed fields', () {
    final snapshot = StatusSnapshot.tryParse(
      jsonEncode({
        'activity': {'active_turns': 'lots', 'completed': 4.0},
        'model': {'fallback_count': null},
        'channels': 'not a list',
        'resources': {'memory_rss_bytes': '88MB'},
      }),
    )!;

    expect(snapshot.activity.activeTurns, 0);
    expect(snapshot.activity.completed, 4);
    expect(snapshot.model.activeModel, '');
    expect(snapshot.model.fallbackCount, 0);
    expect(snapshot.channels, isEmpty);
    expect(snapshot.resources.memoryRssBytes, 0);
  });

  test('ignores any sensitive field the payload should never carry', () {
    // Defence in depth: Core builds the payload from safe DTOs, and the parser
    // reads only known-safe keys, so an unexpected field cannot become state
    // the UI could render.
    final snapshot = StatusSnapshot.tryParse(
      jsonEncode({
        'activity': {
          'active_turns': 1,
          'user_message': 'PROMPT-my-medical-records',
          'session_key': 'SESSIONKEY-448812733',
        },
        'model': {'active_model': 'm', 'api_key': 'sk-should-never-appear'},
        'channels': [
          {'name': 'telegram', 'running': true, 'chat_id': 'CHATID-448812733'},
        ],
      }),
    )!;

    expect(snapshot.activity.activeTurns, 1);
    expect(snapshot.model.activeModel, 'm');
    expect(snapshot.channels.single.name, 'telegram');
    // Nothing else was retained: the classes have no field to hold it.
    expect(snapshot.channels.single.running, isTrue);
  });
}
