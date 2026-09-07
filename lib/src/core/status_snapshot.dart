import 'dart:convert';

/// The operational snapshot shown on the Status screen.
///
/// These classes mirror the Core `pkg/status` DTOs field for field. They carry
/// operational metadata only: counts, states, and display names. No prompt,
/// message, session key, chat id, turn id, tool argument, tool result,
/// credential or path is present in the payload they parse, and none may be
/// added here.
class StatusSnapshot {
  const StatusSnapshot({
    required this.system,
    required this.activity,
    required this.model,
    required this.channels,
    required this.resources,
  });

  final StatusSystem system;
  final StatusActivity activity;
  final StatusModel model;
  final List<StatusChannel> channels;
  final StatusResources resources;

  /// Parses the raw JSON the host returns, or null when it is absent or
  /// malformed.
  ///
  /// Null means "detail unavailable" and the screen says so. Substituting an
  /// empty snapshot would present zeroed counters as though they were real
  /// measurements.
  static StatusSnapshot? tryParse(String? raw) {
    if (raw == null || raw.isEmpty) return null;
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map<String, dynamic>) return null;
      return StatusSnapshot(
        system: StatusSystem._fromJson(_mapOf(decoded['system'])),
        activity: StatusActivity._fromJson(_mapOf(decoded['activity'])),
        model: StatusModel._fromJson(_mapOf(decoded['model'])),
        channels: _channelsOf(decoded['channels']),
        resources: StatusResources._fromJson(_mapOf(decoded['resources'])),
      );
    } catch (_) {
      return null;
    }
  }

  static Map<String, dynamic> _mapOf(Object? value) =>
      value is Map<String, dynamic> ? value : const <String, dynamic>{};

  static List<StatusChannel> _channelsOf(Object? value) {
    if (value is! List) return const <StatusChannel>[];
    return value
        .whereType<Map<String, dynamic>>()
        .map(StatusChannel._fromJson)
        .toList(growable: false);
  }
}

/// Gateway-process facts.
class StatusSystem {
  const StatusSystem({required this.uptimeSeconds});

  /// How long the gateway process has been serving, in whole seconds.
  ///
  /// A number with a stated unit. The legacy /health `uptime` field is Go's
  /// own duration text ("27.707765309s"), which this deliberately does not
  /// read: piping that to the screen is what produced an unreadable,
  /// unlocalizable value. Zero is a real reading — a gateway that started a
  /// moment ago — and is distinct from having no snapshot at all, which is
  /// what "detail unavailable" looks like.
  final int uptimeSeconds;

  factory StatusSystem._fromJson(Map<String, dynamic> json) =>
      StatusSystem(uptimeSeconds: _int(json['uptime_seconds']));
}

/// Turn, subagent and tool activity.
///
/// The three gauges are instantaneous. The five counters and [lastActivity]
/// are cumulative for the life of the gateway process and reset when it
/// restarts, which is why the screen labels them "since Gateway start".
class StatusActivity {
  const StatusActivity({
    required this.activeTurns,
    required this.activeSubagents,
    required this.waiting,
    required this.completed,
    required this.failed,
    required this.cancelled,
    required this.toolCalls,
    required this.toolCallsFailed,
    required this.lastActivity,
  });

  final int activeTurns;
  final int activeSubagents;
  final int waiting;
  final int completed;
  final int failed;
  final int cancelled;
  final int toolCalls;
  final int toolCallsFailed;

  /// When a turn last ended, or null when none has in this gateway process.
  final DateTime? lastActivity;

  factory StatusActivity._fromJson(Map<String, dynamic> json) {
    final unix = _int(json['last_activity_unix']);
    return StatusActivity(
      activeTurns: _int(json['active_turns']),
      activeSubagents: _int(json['active_subagents']),
      waiting: _int(json['waiting']),
      completed: _int(json['completed']),
      failed: _int(json['failed']),
      cancelled: _int(json['cancelled']),
      toolCalls: _int(json['tool_calls']),
      toolCallsFailed: _int(json['tool_calls_failed']),
      // Zero means no turn has ended yet, not 1970.
      lastActivity: unix > 0
          ? DateTime.fromMillisecondsSinceEpoch(unix * 1000)
          : null,
    );
  }
}

/// Which model is answering, and which one is configured.
class StatusModel {
  const StatusModel({
    required this.activeModel,
    required this.configuredModel,
    required this.provider,
    required this.fallbackCount,
  });

  final String activeModel;
  final String configuredModel;
  final String provider;
  final int fallbackCount;

  /// Whether the runtime model has been moved away from the configured
  /// default by the advanced `/switch model to <name>` command.
  ///
  /// The screen shows the configured default only when this is true, so the
  /// ordinary case stays uncluttered and the unusual one is visible.
  bool get hasDivergence =>
      activeModel.isNotEmpty &&
      configuredModel.isNotEmpty &&
      activeModel != configuredModel;

  factory StatusModel._fromJson(Map<String, dynamic> json) => StatusModel(
    activeModel: _string(json['active_model']),
    configuredModel: _string(json['configured_model']),
    provider: _string(json['provider']),
    fallbackCount: _int(json['fallback_count']),
  );
}

/// One configured channel's runtime state.
class StatusChannel {
  const StatusChannel({
    required this.name,
    required this.configured,
    required this.started,
    required this.running,
  });

  final String name;
  final bool configured;
  final bool started;
  final bool running;

  factory StatusChannel._fromJson(Map<String, dynamic> json) => StatusChannel(
    name: _string(json['name']),
    configured: _bool(json['configured']),
    started: _bool(json['started']),
    running: _bool(json['running']),
  );
}

/// The Core process's own resource usage.
class StatusResources {
  const StatusResources({
    required this.memoryRssBytes,
    required this.cpuSeconds,
  });

  final int memoryRssBytes;

  /// Cumulative processor time since the gateway started. Not a percentage:
  /// a percentage would need a sampler, and Status runs none.
  final double cpuSeconds;

  factory StatusResources._fromJson(Map<String, dynamic> json) =>
      StatusResources(
        memoryRssBytes: _int(json['memory_rss_bytes']),
        cpuSeconds: _double(json['cpu_seconds']),
      );
}

int _int(Object? value) {
  if (value is int) return value;
  if (value is num) return value.toInt();
  return 0;
}

double _double(Object? value) {
  if (value is double) return value;
  if (value is num) return value.toDouble();
  return 0;
}

bool _bool(Object? value) => value is bool && value;

String _string(Object? value) => value is String ? value : '';
