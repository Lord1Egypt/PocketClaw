import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/core/status_snapshot.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';

/// The Status screen's operational sections.
///
/// Everything here is presentation over an already-safe payload: the snapshot
/// it renders carries counts, states and display names only, so this widget
/// has nothing sensitive to filter and must never gain the ability to reach
/// past it for more.
///
/// Layout uses the existing Aperture tokens and directional insets throughout,
/// so it mirrors correctly under RTL locales without a second code path.
class StatusSections extends StatelessWidget {
  const StatusSections({
    super.key,
    required this.gatewayRunning,
    required this.uptime,
    required this.appVersion,
    required this.coreVersion,
    required this.snapshot,
  });

  final bool gatewayRunning;
  final String uptime;
  final String appVersion;
  final String coreVersion;

  /// Null when the detailed payload is unavailable — the gateway is stopped,
  /// the host could not authenticate, or the platform does not provide it.
  /// The screen says so rather than showing zeroes as though they were real.
  final StatusSnapshot? snapshot;

  static const String _unavailable = '—';

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final detail = snapshot;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _section(
          context,
          title: l10n.statusSectionSystem,
          icon: Icons.dns_outlined,
          rows: [
            _Row(
              l10n.statusGateway,
              gatewayRunning ? l10n.statusRunning : l10n.statusStopped,
              tone: gatewayRunning ? _Tone.good : _Tone.muted,
            ),
            _Row(l10n.statusUptime, _orDash(uptime)),
            _Row(l10n.statusAppVersion, _orDash(appVersion)),
            _Row(l10n.statusCoreVersion, _orDash(coreVersion)),
          ],
        ),
        if (detail == null)
          _unavailableNotice(context, l10n)
        else ...[
          _section(
            context,
            title: l10n.statusSectionAi,
            icon: Icons.auto_awesome_outlined,
            rows: [
              _Row(l10n.statusActiveModel, _orDash(detail.model.activeModel)),
              // The configured default appears only when the advanced /switch
              // command has moved the runtime model away from it. Showing it
              // always would add noise; hiding it always would conceal a real
              // disagreement between two surfaces.
              if (detail.model.hasDivergence)
                _Row(
                  l10n.statusConfiguredDefault,
                  detail.model.configuredModel,
                  tone: _Tone.notice,
                ),
              _Row(l10n.statusProvider, _orDash(detail.model.provider)),
              _Row(l10n.statusFallbacks, '${detail.model.fallbackCount}'),
              _Row(l10n.statusActiveTurns, '${detail.activity.activeTurns}'),
              _Row(
                l10n.statusActiveSubagents,
                '${detail.activity.activeSubagents}',
              ),
            ],
          ),
          _section(
            context,
            title: l10n.statusSectionActivity,
            icon: Icons.timeline_outlined,
            caption: l10n.statusSinceGatewayStart,
            rows: [
              _Row(l10n.statusWaiting, '${detail.activity.waiting}'),
              _Row(l10n.statusCompleted, '${detail.activity.completed}'),
              _Row(
                l10n.statusFailed,
                '${detail.activity.failed}',
                tone: detail.activity.failed > 0 ? _Tone.bad : _Tone.normal,
              ),
              _Row(l10n.statusCancelled, '${detail.activity.cancelled}'),
              _Row(l10n.statusToolCalls, '${detail.activity.toolCalls}'),
              _Row(
                l10n.statusToolCallsFailed,
                '${detail.activity.toolCallsFailed}',
                tone: detail.activity.toolCallsFailed > 0
                    ? _Tone.bad
                    : _Tone.normal,
              ),
              _Row(
                l10n.statusLastActivity,
                _relativeTime(l10n, detail.activity.lastActivity),
              ),
            ],
          ),
          _section(
            context,
            title: l10n.statusSectionChannels,
            icon: Icons.hub_outlined,
            rows: detail.channels.isEmpty
                ? [_Row(l10n.statusNoChannels, '', tone: _Tone.muted)]
                : detail.channels
                      .map(
                        (channel) => _Row(
                          channel.name,
                          _channelState(l10n, channel),
                          tone: _channelTone(channel),
                        ),
                      )
                      .toList(),
          ),
          _section(
            context,
            title: l10n.statusSectionResources,
            icon: Icons.memory_outlined,
            rows: [
              _Row(
                l10n.statusCoreMemory,
                _formatBytes(detail.resources.memoryRssBytes),
              ),
              _Row(
                l10n.statusCoreCpuTime,
                _formatCpuSeconds(detail.resources.cpuSeconds),
              ),
            ],
          ),
        ],
      ],
    );
  }

  /// A channel is Running only when it both started and reports running.
  ///
  /// `started` is checked first on purpose. The running flag is latched by the
  /// channel itself and Core already refuses to pair it with a missing worker,
  /// but this screen must not be the place where a stale flag becomes a claim:
  /// a channel that never started reads as failed here regardless of what its
  /// own flag says. Anything else configured but not running is stopped.
  static String _channelState(AppLocalizations l10n, StatusChannel channel) {
    if (!channel.started) return l10n.statusChannelFailedToStart;
    if (channel.running) return l10n.statusRunning;
    return l10n.statusStopped;
  }

  static _Tone _channelTone(StatusChannel channel) {
    if (!channel.started) return _Tone.bad;
    if (channel.running) return _Tone.good;
    return _Tone.muted;
  }

  static String _relativeTime(AppLocalizations l10n, DateTime? at) {
    if (at == null) return _unavailable;
    final elapsed = DateTime.now().difference(at);
    if (elapsed.isNegative || elapsed.inMinutes < 1) return l10n.statusJustNow;
    if (elapsed.inHours < 1) return l10n.statusMinutesAgo(elapsed.inMinutes);
    if (elapsed.inDays < 1) return l10n.statusHoursAgo(elapsed.inHours);
    return l10n.statusDaysAgo(elapsed.inDays);
  }

  static String _formatBytes(int bytes) {
    if (bytes <= 0) return _unavailable;
    const mb = 1024 * 1024;
    if (bytes < mb) return '${(bytes / 1024).round()} KB';
    return '${(bytes / mb).round()} MB';
  }

  static String _formatCpuSeconds(double seconds) {
    if (seconds <= 0) return _unavailable;
    final total = seconds.round();
    if (total < 60) return '${total}s';
    final minutes = total ~/ 60;
    if (minutes < 60) return '${minutes}m ${total % 60}s';
    return '${minutes ~/ 60}h ${minutes % 60}m';
  }

  static String _orDash(String value) =>
      value.trim().isEmpty ? _unavailable : value;

  Widget _unavailableNotice(BuildContext context, AppLocalizations l10n) {
    final tokens = context.aperture;
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(bottom: ApertureTheme.spaceMd),
      padding: const EdgeInsets.all(ApertureTheme.spaceMd),
      decoration: BoxDecoration(
        color: tokens.surface1,
        borderRadius: BorderRadius.circular(ApertureTheme.radiusMd),
        border: Border.all(color: tokens.border),
      ),
      child: Row(
        children: [
          Icon(Icons.info_outline, size: 18, color: tokens.textFaint),
          const SizedBox(width: ApertureTheme.spaceSm),
          Expanded(
            child: Text(
              l10n.statusDetailUnavailable,
              style: TextStyle(fontSize: 13, color: tokens.textMuted),
            ),
          ),
        ],
      ),
    );
  }

  Widget _section(
    BuildContext context, {
    required String title,
    required IconData icon,
    required List<_Row> rows,
    String? caption,
  }) {
    final tokens = context.aperture;
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(bottom: ApertureTheme.spaceMd),
      decoration: BoxDecoration(
        color: tokens.surface1,
        borderRadius: BorderRadius.circular(ApertureTheme.radiusMd),
        border: Border.all(color: tokens.border),
      ),
      clipBehavior: Clip.antiAlias,
      child: Padding(
        padding: const EdgeInsets.all(ApertureTheme.spaceLg),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  width: 32,
                  height: 32,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: tokens.surface3,
                    borderRadius: BorderRadius.circular(
                      ApertureTheme.radiusXs,
                    ),
                  ),
                  child: Icon(icon, color: tokens.accent, size: 16),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(
                    title,
                    style: GoogleFonts.inter(
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                      letterSpacing: 0.9,
                      color: tokens.textFaint,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                if (caption != null)
                  Flexible(
                    child: Text(
                      caption,
                      textAlign: TextAlign.end,
                      style: TextStyle(fontSize: 11, color: tokens.textFaint),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
              ],
            ),
            const SizedBox(height: ApertureTheme.spaceMd),
            for (final row in rows) _row(context, row),
          ],
        ),
      ),
    );
  }

  Widget _row(BuildContext context, _Row row) {
    final tokens = context.aperture;
    final valueColor = switch (row.tone) {
      _Tone.good => tokens.success,
      _Tone.bad => tokens.danger,
      _Tone.notice => tokens.warning,
      _Tone.muted => tokens.textFaint,
      _Tone.normal => tokens.text,
    };

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 5),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            child: Text(
              row.label,
              style: TextStyle(fontSize: 13, color: tokens.textMuted),
            ),
          ),
          const SizedBox(width: ApertureTheme.spaceMd),
          Flexible(
            child: Text(
              row.value,
              textAlign: TextAlign.end,
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w600,
                color: valueColor,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }
}

enum _Tone { normal, good, bad, notice, muted }

class _Row {
  const _Row(this.label, this.value, {this.tone = _Tone.normal});

  final String label;
  final String value;
  final _Tone tone;
}
