import 'dart:async';
import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:pocketclaw/src/core/legacy_workspace.dart';
import 'package:pocketclaw/src/core/pocketclaw_channel.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/core/app_theme.dart';
import 'package:pocketclaw/src/ui/github_settings_card.dart';
import 'package:pocketclaw/src/ui/whats_new_page.dart';
import 'package:pocketclaw/src/whats_new/whats_new_release.dart';
import 'package:pocketclaw/src/whats_new/whats_new_seen_store.dart';
import 'context_memory_card.dart';
import 'models_settings_card.dart';
import 'package:pocketclaw/src/ui/telegram_settings_card.dart';

const String _aboutProjectName = 'PocketClaw';

class AboutInfo {
  const AboutInfo({required this.appVersion, required this.coreVersion});

  /// PC-DEF-063. Null means the probe failed, which is a different thing from
  /// "not read yet" -- that one is the future not having completed. Keeping
  /// them apart is what stopped Unknown being used as a loading placeholder.

  final String appVersion;
  final String? coreVersion;
}

/// Settings control for the two launch auto-start preferences.
///
/// The preference and the runtime are shown as two separate lines on purpose:
/// a preference that is ON does not mean the component is running, and a
/// component the user stopped by hand stays stopped until the next app launch.
/// A group label above a run of Settings cards.
///
/// Uppercase is used at exactly one size in Aperture, and this is it. The
/// label sizes to its content and wraps, so a long German or Russian
/// translation lengthens the block instead of clipping.
/// The Settings header actions: ghost buttons on a hairline, never filled.
/// Only a card's single primary action earns the accent fill.
ButtonStyle _ghostActionStyle(BuildContext context) {
  final tokens = context.aperture;
  return ElevatedButton.styleFrom(
    backgroundColor: tokens.surface1,
    foregroundColor: tokens.text,
    elevation: 0,
    minimumSize: const Size(0, ApertureTheme.minTouchTarget),
    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
    shape: RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(ApertureTheme.radiusSm),
      side: BorderSide(color: tokens.border),
    ),
  );
}

class SettingsSectionLabel extends StatelessWidget {
  const SettingsSectionLabel(this.label, {super.key});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsetsDirectional.only(
        start: 4,
        top: ApertureTheme.spaceSm,
        bottom: ApertureTheme.spaceSm,
      ),
      child: Text(
        label.toUpperCase(),
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
          color: context.aperture.textFaint,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.9,
          height: 1.2,
        ),
      ),
    );
  }
}

class AutoStartSettingsCard extends StatelessWidget {
  const AutoStartSettingsCard({
    super.key,
    required this.serviceEnabled,
    required this.gatewayEnabled,
    required this.serviceStatus,
    required this.onServiceChanged,
    required this.onGatewayChanged,
  });

  final bool serviceEnabled;
  final bool gatewayEnabled;
  final ServiceStatus serviceStatus;
  final ValueChanged<bool> onServiceChanged;
  final ValueChanged<bool> onGatewayChanged;

  static String runtimeLabel(AppLocalizations l10n, ServiceStatus status) =>
      switch (status) {
        ServiceStatus.running => l10n.runtimeRunning,
        ServiceStatus.starting => l10n.runtimeStarting,
        ServiceStatus.stopped => l10n.runtimeStopped,
      };

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Card(
      margin: EdgeInsets.zero,
      child: Column(
        children: [
          SwitchListTile.adaptive(
            title: Text(l10n.autoStartServiceTitle),
            isThreeLine: true,
            subtitle: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  serviceEnabled
                      ? l10n.autoStartPreferenceOn
                      : l10n.autoStartPreferenceOff,
                ),
                const SizedBox(height: 4),
                _RuntimeStatusChip(
                  label: runtimeLabel(l10n, serviceStatus),
                  status: serviceStatus,
                ),
              ],
            ),
            value: serviceEnabled,
            onChanged: onServiceChanged,
          ),
          const Divider(height: 1),
          SwitchListTile.adaptive(
            title: Text(l10n.autoStartGatewayTitle),
            isThreeLine: true,
            subtitle: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  gatewayEnabled
                      ? l10n.autoStartPreferenceOn
                      : l10n.autoStartPreferenceOff,
                ),
                const SizedBox(height: 2),
                Text(l10n.gatewayAutoStartHint),
              ],
            ),
            value: gatewayEnabled,
            onChanged: onGatewayChanged,
          ),
        ],
      ),
    );
  }
}

/// The runtime half of an Auto-Start row: a dot and the word it already said.
///
/// Running is live machine state, which is the one thing Signal is reserved
/// for. The label is always present — the dot never carries the meaning alone.
class _RuntimeStatusChip extends StatelessWidget {
  const _RuntimeStatusChip({required this.label, required this.status});

  final String label;
  final ServiceStatus status;

  @override
  Widget build(BuildContext context) {
    final tokens = context.aperture;
    final color = switch (status) {
      ServiceStatus.running => tokens.signal,
      ServiceStatus.starting => tokens.warning,
      ServiceStatus.stopped => tokens.textFaint,
    };
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 6,
          height: 6,
          margin: const EdgeInsetsDirectional.only(end: 6),
          decoration: BoxDecoration(color: color, shape: BoxShape.circle),
        ),
        Flexible(
          child: Text(
            label,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
              color: color,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ],
    );
  }
}

class ConfigPage extends StatefulWidget {
  final Future<AboutInfo> Function()? aboutInfoLoader;
  final Future<void> Function(String path)? onManageTelegram;

  /// Opens a console route in the embedded Dashboard. Same mechanism as
  /// [onManageTelegram]; named generically because more than one card uses it.
  final Future<void> Function(String path)? onManageConsole;

  /// Where the "already read the notes" mark is kept. Injectable so a test can
  /// drive the badge without a platform preference store.
  final WhatsNewSeenStore? whatsNewSeenStore;

  /// Resolves the release identity — the versionName, never the versionCode.
  final Future<String> Function()? whatsNewVersionLoader;

  const ConfigPage({
    super.key,
    this.aboutInfoLoader,
    this.onManageTelegram,
    this.onManageConsole,
    this.whatsNewSeenStore,
    this.whatsNewVersionLoader,
  });

  @override
  State<ConfigPage> createState() => ConfigPageState();
}

class ConfigPageState extends State<ConfigPage> with WidgetsBindingObserver {
  /// What the host says about PocketClaw's permission to post notifications.
  /// Null until the first read completes. PC-DEF-058.
  NotificationPermissionStatus? _notificationPermission;
  static ConfigPageState? _current;
  static ConfigPageState? get current => _current;
  final _publicAddressController = TextEditingController();
  // Focus nodes for TV navigation
  final _whatsNewFocusNode = FocusNode();
  final _aboutFocusNode = FocusNode();
  final _contextMemoryFocusNode = FocusNode();
  final _publicModeFocusNode = FocusNode();
  final _publicAddressFocusNode = FocusNode();
  final _languageFocusNode = FocusNode();
  final List<FocusNode> _themeFocusNodes = [];

  /// PC-DEF-077. A workspace an older install left in `Download/pocketclaw`,
  /// offered for an explicit copy until the owner copies it or hides the notice.
  LegacyWorkspaceStatus _legacyWorkspace = LegacyWorkspaceStatus.none;
  bool _legacyWorkspaceBusy = false;
  static const _legacyWorkspaceNoticeHiddenKey =
      'legacy_workspace_notice_hidden';

  /// The release the notes belong to, and whether the user has read them.
  String? _whatsNewVersion;
  bool _whatsNewUnseen = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _current = this;

    // Initialize theme focus nodes
    _themeFocusNodes.addAll(
      List.generate(AppThemeMode.values.length, (_) => FocusNode()),
    );

    _loadWhatsNewState();
    unawaited(_refreshLegacyWorkspace());
    // PC-DEF-058. Ask for notification permission once, after the first frame so
    // the app is on screen behind the system dialog rather than the dialog being
    // the first thing a fresh install shows. The host refuses to ask twice, so
    // this cannot nag.
    WidgetsBinding.instance.addPostFrameCallback(
      (_) => unawaited(_ensureNotificationPermission()),
    );
  }

  /// Requests notification permission on first run, and records the outcome so
  /// Settings can show it.
  Future<void> _ensureNotificationPermission() async {
    var status = await PocketClawChannel.getNotificationPermission();
    if (status.shouldRequest) {
      status = await PocketClawChannel.requestNotificationPermission();
    }
    if (!mounted) return;
    setState(() => _notificationPermission = status);
  }

  /// Re-reads the state, for the case where the user granted it in Settings and
  /// came back.
  Future<void> _refreshNotificationPermission() async {
    final status = await PocketClawChannel.getNotificationPermission();
    if (!mounted) return;
    setState(() => _notificationPermission = status);
  }

  Future<void> _refreshLegacyWorkspace() async {
    if (!Platform.isAndroid) return;
    final prefs = await SharedPreferences.getInstance();
    var status = LegacyWorkspaceStatus.none;
    if (!(prefs.getBool(_legacyWorkspaceNoticeHiddenKey) ?? false)) {
      try {
        status = await PocketClawChannel.getLegacyWorkspaceStatus();
      } catch (_) {
        status = LegacyWorkspaceStatus.none;
      }
    }
    if (!mounted) return;
    setState(() => _legacyWorkspace = status);
  }

  Future<void> _hideLegacyWorkspaceNotice() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool(_legacyWorkspaceNoticeHiddenKey, true);
    if (!mounted) return;
    setState(() => _legacyWorkspace = LegacyWorkspaceStatus.none);
  }

  Future<void> _importLegacyWorkspace() async {
    setState(() => _legacyWorkspaceBusy = true);
    LegacyWorkspaceImportResult result;
    try {
      result = await PocketClawChannel.importLegacyWorkspace();
    } catch (_) {
      result = const LegacyWorkspaceImportResult(
        status: LegacyWorkspaceImportStatus.failed,
      );
    }
    if (!mounted) return;
    setState(() => _legacyWorkspaceBusy = false);
    final l10n = AppLocalizations.of(context)!;
    final message = switch (result.status) {
      LegacyWorkspaceImportStatus.copied => l10n.legacyWorkspaceCopied(
        result.files,
        result.folder,
      ),
      LegacyWorkspaceImportStatus.partial => l10n.legacyWorkspacePartial(
        result.files,
        result.folder,
        result.failed,
      ),
      LegacyWorkspaceImportStatus.failed ||
      LegacyWorkspaceImportStatus.unavailable => l10n.legacyWorkspaceFailed,
      LegacyWorkspaceImportStatus.empty => l10n.legacyWorkspaceEmpty,
      LegacyWorkspaceImportStatus.cancelled ||
      LegacyWorkspaceImportStatus.busy => null,
    };
    if (message != null) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(message)));
    }
    if (result.copiedAnything) await _hideLegacyWorkspaceNotice();
  }

  WhatsNewSeenStore get _whatsNewStore =>
      widget.whatsNewSeenStore ?? const SharedPreferencesWhatsNewSeenStore();

  Future<void> _loadWhatsNewState() async {
    final version =
        await (widget.whatsNewVersionLoader ?? readWhatsNewReleaseVersion)();
    final lastSeen = await _whatsNewStore.readLastSeenVersion();
    if (!mounted) return;
    setState(() {
      _whatsNewVersion = version;
      _whatsNewUnseen = lastSeen != version;
    });
  }

  /// Opens the release notes, then records the release as read.
  ///
  /// The mark is written once the route is on screen and never on the way in
  /// to Settings, so the badge survives a user who only passed through.
  Future<void> _openWhatsNew() async {
    final version = _whatsNewVersion ?? currentWhatsNewRelease.version;
    final navigator = Navigator.of(context);
    final closed = navigator.push(
      MaterialPageRoute<void>(builder: (_) => const WhatsNewPage()),
    );

    await _whatsNewStore.writeLastSeenVersion(version);
    if (mounted) {
      setState(() => _whatsNewUnseen = false);
    }

    await closed;
    if (mounted) {
      _whatsNewFocusNode.requestFocus();
    }
  }

  String _getLanguageName(String code) {
    return switch (code) {
      'ar' => 'العربية',
      'de' => 'Deutsch',
      'en' => 'English',
      'es' => 'Español',
      'fr' => 'Français',
      'hi' => 'हिन्दी',
      'id' => 'Bahasa Indonesia',
      'ja' => '日本語',
      'ko' => '한국어',
      'pt' => 'Português',
      'ru' => 'Русский',
      'zh' => '中文',
      _ => code,
    };
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _current = null;
    _publicAddressController.dispose();

    _whatsNewFocusNode.dispose();
    _aboutFocusNode.dispose();
    _contextMemoryFocusNode.dispose();
    _publicModeFocusNode.dispose();
    _publicAddressFocusNode.dispose();
    _languageFocusNode.dispose();
    for (final node in _themeFocusNodes) {
      node.dispose();
    }

    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    super.didChangeAppLifecycleState(state);
    if (state == AppLifecycleState.resumed && Platform.isAndroid) {
      unawaited(_refreshLegacyWorkspace());
    }
  }

  Future<void> _togglePublicMode(bool value) async {
    final service = context.read<ServiceManager>();

    final applied = await service.applyPublicMode(value);

    if (!mounted) return;
    if (!applied && service.publicModeApplyError != null) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(service.publicModeApplyError!)));
    }
  }

  void _togglePublicModeFromFocus() {
    final service = context.read<ServiceManager>();
    _togglePublicMode(!service.publicMode);
  }

  Future<AboutInfo> _loadAboutInfo() async {
    final service = context.read<ServiceManager>();
    final appVersion = await service.getAppVersion();
    final coreVersion = await service.getCoreVersion();
    return AboutInfo(appVersion: appVersion, coreVersion: coreVersion);
  }

  /// The text for a version that has already been probed.
  ///
  /// Only reached once the probe has finished, so an empty or absent value here
  /// means it genuinely failed -- which is the one case that may say
  /// unavailable. While the probe is still running the caller renders the
  /// pending state instead.
  String _normalizeAboutVersion(String? value, AppLocalizations l10n) {
    final normalized = value?.trim() ?? '';
    if (normalized.isEmpty || normalized.toLowerCase() == 'unknown') {
      return l10n.aboutVersionUnavailable;
    }
    return normalized;
  }

  /// One labelled version line.
  ///
  /// Label above value rather than beside it: the old layout reserved a fixed
  /// 148px label column, which a longer translation overflows and a narrow
  /// phone cannot afford. Stacking has no width to get wrong, and it mirrors
  /// for Arabic without a second layout.
  Widget _buildAboutVersionRow(
    BuildContext context, {
    required String label,
    required Widget value,
  }) {
    final textTheme = Theme.of(context).textTheme;
    final tokens = context.aperture;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: textTheme.bodySmall?.copyWith(
            color: tokens.textMuted,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: ApertureTheme.spaceXs),
        value,
      ],
    );
  }

  Widget _buildAboutVersionValue(BuildContext context, String value) {
    final textTheme = Theme.of(context).textTheme;
    final tokens = context.aperture;
    // A version, a commit and a build stamp are values a human compares
    // character by character, which is exactly what mono is for. They
    // stay left-to-right in Arabic: a reversed digest is a bug, not
    // localization.
    return Directionality(
      textDirection: TextDirection.ltr,
      child: Text(
        value,
        style: textTheme.bodyMedium?.copyWith(
          fontFamily: 'monospace',
          fontFamilyFallback: const ['Menlo', 'Consolas'],
          color: tokens.text,
        ),
      ),
    );
  }

  /// The value slot while the versions are still being read.
  ///
  /// It occupies one line, the same as the value that replaces it, so the
  /// dialog does not resize when the future resolves.
  Widget _buildAboutVersionPending(BuildContext context) {
    final lineHeight = Theme.of(context).textTheme.bodyMedium?.fontSize ?? 14.0;
    return SizedBox(
      height: lineHeight * 1.4,
      child: Align(
        alignment: AlignmentDirectional.centerStart,
        child: const SizedBox(
          width: 14,
          height: 14,
          child: CircularProgressIndicator(strokeWidth: 2),
        ),
      ),
    );
  }

  Widget _buildAboutIdentity(BuildContext context) {
    final textTheme = Theme.of(context).textTheme;
    final tokens = context.aperture;
    return Row(
      children: [
        Container(
          width: 40,
          height: 40,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: tokens.accentSoft,
            borderRadius: BorderRadius.circular(ApertureTheme.radiusSm),
          ),
          child: Icon(Icons.camera, size: 22, color: tokens.accent),
        ),
        const SizedBox(width: ApertureTheme.spaceMd),
        Expanded(
          child: Text(
            _aboutProjectName,
            style: textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
          ),
        ),
      ],
    );
  }

  Future<void> _showAboutDialog() async {
    final l10n = AppLocalizations.of(context)!;
    final aboutInfoFuture = widget.aboutInfoLoader?.call() ?? _loadAboutInfo();

    await showDialog<void>(
      context: context,
      builder: (ctx) {
        final tokens = ctx.aperture;
        return AlertDialog(
          title: Text(l10n.about),
          content: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 420),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _buildAboutIdentity(ctx),
                const SizedBox(height: ApertureTheme.spaceMd),
                Text(
                  l10n.aboutDescription,
                  style: Theme.of(
                    ctx,
                  ).textTheme.bodyMedium?.copyWith(color: tokens.textMuted),
                ),
                const SizedBox(height: ApertureTheme.spaceMd),
                // The versions are grouped into one card so they read as a
                // block of provenance rather than as two more sentences.
                ApertureBracket(
                  fill: tokens.surface1,
                  borderColor: tokens.border,
                  borderWidth: 1,
                  padding: const EdgeInsetsDirectional.all(
                    ApertureTheme.spaceMd,
                  ),
                  child: FutureBuilder<AboutInfo>(
                    future: aboutInfoFuture,
                    builder: (ctx, snapshot) {
                      // Loading is decided by the future, not by the value:
                      // an unread version renders as pending, and only a
                      // completed probe may say unavailable (PC-DEF-063).
                      final loading =
                          snapshot.connectionState == ConnectionState.waiting;
                      final info = snapshot.data;
                      Widget valueFor(String? raw) => loading
                          ? _buildAboutVersionPending(ctx)
                          : _buildAboutVersionValue(
                              ctx,
                              _normalizeAboutVersion(raw, l10n),
                            );
                      return Column(
                        mainAxisSize: MainAxisSize.min,
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          _buildAboutVersionRow(
                            ctx,
                            label: l10n.aboutAppVersionLabel,
                            value: valueFor(info?.appVersion),
                          ),
                          const SizedBox(height: ApertureTheme.spaceMd),
                          _buildAboutVersionRow(
                            ctx,
                            label: l10n.aboutCoreVersionLabel,
                            value: valueFor(info?.coreVersion),
                          ),
                        ],
                      );
                    },
                  ),
                ),
              ],
            ),
          ),
          actions: [
            FilledButton(
              onPressed: () => Navigator.of(ctx).pop(),
              child: Text(l10n.close),
            ),
          ],
        );
      },
    );

    if (mounted) {
      _aboutFocusNode.requestFocus();
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    // Use scoped Selectors below to avoid whole-page rebuilds when ServiceManager changes.

    return FocusTraversalGroup(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // A Wrap, not a Row: the title and the actions each keep their
            // natural width and drop to a second line when the two cannot
            // share one. A Row gave the actions their full width first and
            // left the title whatever remained, which with the unseen NEW
            // badge showing was nothing at all.
            Wrap(
              alignment: WrapAlignment.spaceBetween,
              crossAxisAlignment: WrapCrossAlignment.center,
              spacing: 12,
              runSpacing: 12,
              children: [
                Text(
                  l10n.settings,
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                Wrap(
                  crossAxisAlignment: WrapCrossAlignment.center,
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    Tooltip(
                      message: l10n.whatsNewDescription,
                      child: FocusableButton(
                        focusNode: _whatsNewFocusNode,
                        onPressed: _openWhatsNew,
                        prevFocusNode: _whatsNewFocusNode,
                        nextFocusNode: _aboutFocusNode,
                        style: _ghostActionStyle(context),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            const Icon(Icons.auto_awesome_outlined, size: 18),
                            const SizedBox(width: 6),
                            Flexible(
                              child: Text(
                                l10n.whatsNewTitle,
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                            if (_whatsNewUnseen) ...[
                              const SizedBox(width: 6),
                              _WhatsNewBadge(label: l10n.whatsNewBadge),
                            ],
                          ],
                        ),
                      ),
                    ),
                    Tooltip(
                      message: l10n.about,
                      child: FocusableButton(
                        focusNode: _aboutFocusNode,
                        onPressed: () {
                          _showAboutDialog();
                        },
                        prevFocusNode: _whatsNewFocusNode,
                        nextFocusNode: _publicModeFocusNode,
                        style: _ghostActionStyle(context),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            const Icon(Icons.info_outline, size: 18),
                            const SizedBox(width: 6),
                            Text(l10n.about),
                          ],
                        ),
                      ),
                    ),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 8),

            SettingsSectionLabel(l10n.settingsGroupConnection),

            Selector<ServiceManager, ({bool isPublic, bool isApplying})>(
              selector: (_, s) =>
                  (isPublic: s.publicMode, isApplying: s.isApplyingPublicMode),
              builder: (_, publicModeState, _) => PublicModeToggle(
                focusNode: _publicModeFocusNode,
                isPublicMode: publicModeState.isPublic,
                isApplying: publicModeState.isApplying,
                onToggle: _togglePublicModeFromFocus,
                onArrowDown: () => _contextMemoryFocusNode.requestFocus(),
                onArrowUp: () => _aboutFocusNode.requestFocus(),
              ),
            ),
            const SizedBox(height: 16),

            // Where another device reaches the Dashboard, shown only while
            // Public Mode is on. The editable host and the port that used to
            // sit here were desktop launcher settings: the Android host binds
            // 127.0.0.1 or all interfaces from Public Mode alone, always on
            // port 18800. PC-DEF-083.
            Selector<ServiceManager, ({bool isPublic, String? url})>(
              selector: (_, s) =>
                  (isPublic: s.publicMode, url: s.publicDashboardUrl),
              builder: (_, addressState, _) {
                if (!addressState.isPublic) return const SizedBox.shrink();
                _publicAddressController.text =
                    addressState.url ?? l10n.unableToGetDeviceIp;
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l10n.address,
                      style: Theme.of(context).textTheme.labelLarge?.copyWith(
                        color: context.aperture.textMuted,
                      ),
                    ),
                    const SizedBox(height: 8),
                    FocusableTextField(
                      controller: _publicAddressController,
                      focusNode: _publicAddressFocusNode,
                      label: l10n.address,
                      enabled: false,
                      nextFocusNode: _contextMemoryFocusNode,
                      prevFocusNode: _publicModeFocusNode,
                    ),
                    const SizedBox(height: 16),
                  ],
                );
              },
            ),

            if (Platform.isAndroid) ...[
              Selector<
                ServiceManager,
                ({
                  bool serviceEnabled,
                  bool gatewayEnabled,
                  ServiceStatus serviceStatus,
                })
              >(
                selector: (_, s) => (
                  serviceEnabled: s.serviceLaunchAutoStart,
                  gatewayEnabled: s.gatewayLaunchAutoStart,
                  serviceStatus: s.status,
                ),
                builder: (context, autoStart, _) => AutoStartSettingsCard(
                  serviceEnabled: autoStart.serviceEnabled,
                  gatewayEnabled: autoStart.gatewayEnabled,
                  serviceStatus: autoStart.serviceStatus,
                  onServiceChanged: (value) => context
                      .read<ServiceManager>()
                      .setServiceLaunchAutoStart(value),
                  onGatewayChanged: (value) => context
                      .read<ServiceManager>()
                      .setGatewayLaunchAutoStart(value),
                ),
              ),
              const SizedBox(height: 16),
            ],

            // PC-DEF-058. The state is shown whatever it is, and a recovery
            // action appears only when one is needed -- someone who granted the
            // permission has nothing to do here.
            if (Platform.isAndroid && _notificationPermission != null)
              _NotificationPermissionTile(
                status: _notificationPermission!,
                onOpenSettings: () async {
                  await PocketClawChannel.openNotificationSettings();
                  await _refreshNotificationPermission();
                },
              ),
            if (Platform.isAndroid && _notificationPermission != null)
              const SizedBox(height: 16),

            SettingsSectionLabel(l10n.settingsGroupIntegrations),
            TelegramSettingsCard(onManage: widget.onManageTelegram),
            const SizedBox(height: 24),

            SettingsSectionLabel(l10n.settingsGroupAgent),
            ContextMemoryCard(focusNode: _contextMemoryFocusNode),
            const SizedBox(height: 12),
            ModelsSettingsCard(onManage: widget.onManageConsole),
            const SizedBox(height: 12),
            GitHubSettingsCard(
              onCredentialChanged: () =>
                  context.read<ServiceManager>().applyCredentialChange(),
            ),
            const SizedBox(height: 24),

            if (Platform.isAndroid) ...[
              Text(
                l10n.workspaceDirectory,
                style: Theme.of(context).textTheme.titleMedium?.copyWith(
                  color: Theme.of(context).colorScheme.onSurface.withAlpha(153),
                ),
              ),
              const SizedBox(height: 8),
              Container(
                width: double.infinity,
                padding: const EdgeInsets.symmetric(
                  horizontal: 12,
                  vertical: 14,
                ),
                decoration: BoxDecoration(
                  color: Theme.of(context).colorScheme.surface,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: Theme.of(context).colorScheme.outline.withAlpha(60),
                  ),
                ),
                child: Selector<ServiceManager, String>(
                  selector: (_, s) => s.workspacePath,
                  builder: (_, path, _) => Text(
                    path,
                    textDirection: TextDirection.ltr,
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                ),
              ),
              if (_legacyWorkspace.visible) ...[
                const SizedBox(height: 12),
                _LegacyWorkspaceTile(
                  path: _legacyWorkspace.path,
                  busy: _legacyWorkspaceBusy,
                  onImport: _importLegacyWorkspace,
                  onHide: _hideLegacyWorkspaceNotice,
                ),
              ],
            ],
            const SizedBox(height: 24),
            Text(
              l10n.language,
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                color: Theme.of(context).colorScheme.onSurface.withAlpha(153),
              ),
            ),
            const SizedBox(height: 8),
            Selector<ServiceManager, Locale>(
              selector: (_, s) => s.currentLocale,
              builder: (_, currentLocale, _) {
                final service = context.read<ServiceManager>();
                return FocusableButton(
                  focusNode: _languageFocusNode,
                  onPressed: () {},
                  prevFocusNode: _publicModeFocusNode,
                  nextFocusNode: _themeFocusNodes.isNotEmpty
                      ? _themeFocusNodes.first
                      : _languageFocusNode,
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Theme.of(context).colorScheme.surface,
                    foregroundColor: Theme.of(context).colorScheme.onSurface,
                    padding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 12,
                    ),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8),
                      side: BorderSide(
                        color: Theme.of(
                          context,
                        ).colorScheme.outline.withAlpha(60),
                      ),
                    ),
                  ),
                  child: PopupMenuButton<Locale>(
                    initialValue: currentLocale,
                    tooltip: l10n.selectLanguage,
                    onSelected: (locale) => service.setLocale(locale),
                    itemBuilder: (ctx) => AppLocalizations.supportedLocales
                        .map(
                          (locale) => PopupMenuItem(
                            value: locale,
                            child: Text(_getLanguageName(locale.languageCode)),
                          ),
                        )
                        .toList(),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text(_getLanguageName(currentLocale.languageCode)),
                        const SizedBox(width: 8),
                        const Icon(Icons.arrow_drop_down, size: 20),
                      ],
                    ),
                  ),
                );
              },
            ),
            const SizedBox(height: 16),
            const Divider(),
            const SizedBox(height: 8),
            SettingsSectionLabel(l10n.settingsGroupAppearance),
            Text(
              l10n.themeSelection,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 12),
            ThemeModeSelector(
              themeFocusNodes: _themeFocusNodes,
              previousFocusNode: _languageFocusNode,
            ),
            const SizedBox(height: 32),
          ],
        ),
      ),
    );
  }
}

// Focusable widgets with shadow effects

class FocusableTextField extends StatefulWidget {
  final TextEditingController controller;
  final FocusNode focusNode;
  final String label;
  final String? hint;
  final TextInputType? keyboardType;
  final bool enabled;
  final FocusNode nextFocusNode;
  final FocusNode prevFocusNode;
  final VoidCallback? onSubmitted;

  const FocusableTextField({
    super.key,
    required this.controller,
    required this.focusNode,
    required this.label,
    this.hint,
    this.keyboardType,
    this.enabled = true,
    required this.nextFocusNode,
    required this.prevFocusNode,
    this.onSubmitted,
  });

  @override
  State<FocusableTextField> createState() => _FocusableTextFieldState();
}

class _FocusableTextFieldState extends State<FocusableTextField> {
  bool _isFocused = false;

  @override
  void initState() {
    super.initState();
    widget.focusNode.addListener(_onFocusChange);
  }

  @override
  void dispose() {
    widget.focusNode.removeListener(_onFocusChange);
    super.dispose();
  }

  void _onFocusChange() {
    if (mounted && _isFocused != widget.focusNode.hasFocus) {
      setState(() {
        _isFocused = widget.focusNode.hasFocus;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return AnimatedContainer(
      duration: const Duration(milliseconds: 150),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: _isFocused
              ? colorScheme.secondary
              : colorScheme.outline.withAlpha(60),
          width: _isFocused ? 2 : 1,
        ),
      ),
      child: Focus(
        focusNode: widget.focusNode,
        onKeyEvent: (node, event) {
          if (event is KeyDownEvent) {
            if (event.logicalKey == LogicalKeyboardKey.arrowDown) {
              widget.nextFocusNode.requestFocus();
              return KeyEventResult.handled;
            } else if (event.logicalKey == LogicalKeyboardKey.arrowUp) {
              widget.prevFocusNode.requestFocus();
              return KeyEventResult.handled;
            }
          }
          return KeyEventResult.ignored;
        },
        child: TextField(
          controller: widget.controller,
          decoration: InputDecoration(
            labelText: widget.label,
            hintText: widget.hint,
            hintStyle: TextStyle(color: colorScheme.onSurface.withAlpha(153)),
            floatingLabelBehavior: FloatingLabelBehavior.never,
            filled: true,
            fillColor: _isFocused
                ? colorScheme.primaryContainer.withAlpha(40)
                : colorScheme.surface,

            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide: BorderSide(
                color: colorScheme.outline.withValues(alpha: 0.5),
                width: 0,
              ),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
              borderSide: BorderSide(color: colorScheme.primary, width: 0),
            ),
          ),
          keyboardType: widget.keyboardType,
          enabled: widget.enabled,
          onEditingComplete: () {
            widget.onSubmitted?.call();
            widget.nextFocusNode.requestFocus();
          },
          onSubmitted: (_) {
            widget.onSubmitted?.call();
          },
          textInputAction: TextInputAction.next,
        ),
      ),
    );
  }
}

/// The unread marker on the Settings entry. Shown only until the user opens
/// the release notes for the installed versionName.
class _WhatsNewBadge extends StatelessWidget {
  const _WhatsNewBadge({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final tokens = context.aperture;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
      decoration: BoxDecoration(
        color: tokens.accent,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label.toUpperCase(),
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
          color: tokens.accentInk,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.7,
          height: 1.2,
        ),
      ),
    );
  }
}

class FocusableButton extends StatefulWidget {
  final VoidCallback onPressed;
  final FocusNode focusNode;
  final Widget child;
  final ButtonStyle? style;
  final FocusNode nextFocusNode;
  final FocusNode prevFocusNode;

  const FocusableButton({
    super.key,
    required this.onPressed,
    required this.focusNode,
    required this.child,
    this.style,
    required this.nextFocusNode,
    required this.prevFocusNode,
  });

  @override
  State<FocusableButton> createState() => _FocusableButtonState();
}

class _FocusableButtonState extends State<FocusableButton> {
  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return Focus(
      focusNode: widget.focusNode,
      onKeyEvent: (node, event) {
        if (event is KeyDownEvent) {
          if (event.logicalKey == LogicalKeyboardKey.arrowDown) {
            widget.nextFocusNode.requestFocus();
            return KeyEventResult.handled;
          } else if (event.logicalKey == LogicalKeyboardKey.arrowUp) {
            widget.prevFocusNode.requestFocus();
            return KeyEventResult.handled;
          } else if (event.logicalKey == LogicalKeyboardKey.select ||
              event.logicalKey == LogicalKeyboardKey.enter) {
            widget.onPressed();
            return KeyEventResult.handled;
          }
        }
        return KeyEventResult.ignored;
      },
      child: Builder(
        builder: (context) {
          final isFocused = Focus.of(context).hasFocus;
          return AnimatedScale(
            scale: isFocused ? 1.05 : 1.0,
            duration: const Duration(milliseconds: 150),
            child: AnimatedContainer(
              duration: const Duration(milliseconds: 150),
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(10),
                boxShadow: isFocused
                    ? [
                        BoxShadow(
                          color: colorScheme.primary.withValues(alpha: 0),
                          blurRadius: 10,
                          spreadRadius: 2,
                        ),
                      ]
                    : [],
              ),
              child: ElevatedButton(
                onPressed: widget.onPressed,
                style: widget.style?.copyWith(
                  side: WidgetStateProperty.all(
                    isFocused
                        ? BorderSide(color: colorScheme.primary, width: 1)
                        : BorderSide.none,
                  ),
                ),
                child: widget.child,
              ),
            ),
          );
        },
      ),
    );
  }
}

// Public mode toggle widget with proper focus handling
class PublicModeToggle extends StatefulWidget {
  final FocusNode focusNode;
  final bool isPublicMode;
  final bool isApplying;
  final VoidCallback onToggle;
  final VoidCallback onArrowDown;
  final VoidCallback onArrowUp;

  const PublicModeToggle({
    super.key,
    required this.focusNode,
    required this.isPublicMode,
    required this.isApplying,
    required this.onToggle,
    required this.onArrowDown,
    required this.onArrowUp,
  });

  @override
  State<PublicModeToggle> createState() => _PublicModeToggleState();
}

class _PublicModeToggleState extends State<PublicModeToggle> {
  bool _isFocused = false;

  @override
  void initState() {
    super.initState();
    widget.focusNode.addListener(_onFocusChange);
  }

  @override
  void dispose() {
    widget.focusNode.removeListener(_onFocusChange);
    super.dispose();
  }

  void _onFocusChange() {
    if (mounted) {
      setState(() {
        _isFocused = widget.focusNode.hasFocus;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final tokens = context.aperture;

    return Focus(
      focusNode: widget.focusNode,
      canRequestFocus: true,
      descendantsAreFocusable: false,
      onKeyEvent: (node, event) {
        if (event is KeyDownEvent) {
          if (event.logicalKey == LogicalKeyboardKey.arrowDown) {
            widget.onArrowDown();
            return KeyEventResult.handled;
          } else if (event.logicalKey == LogicalKeyboardKey.arrowUp) {
            widget.onArrowUp();
            return KeyEventResult.handled;
          } else if (event.logicalKey == LogicalKeyboardKey.select ||
              event.logicalKey == LogicalKeyboardKey.enter) {
            if (!widget.isApplying) widget.onToggle();
            return KeyEventResult.handled;
          }
        }
        return KeyEventResult.ignored;
      },
      child: GestureDetector(
        onTap: widget.isApplying ? null : widget.onToggle,
        // Public Mode exposes the Dashboard on the LAN. Warning colour is the
        // honest one for network exposure, and it is deliberately not the
        // accent — this is the one card allowed to differ.
        child: ApertureBracket(
          fill: widget.isPublicMode
              ? tokens.warning.withValues(alpha: 0.12)
              : tokens.surface1,
          bracket: widget.isPublicMode ? tokens.warning : null,
          borderColor: _isFocused ? tokens.accent : tokens.border,
          borderWidth: _isFocused ? 2 : 1,
          padding: const EdgeInsetsDirectional.only(
            start: 16,
            end: 16,
            top: 12,
            bottom: 12,
          ),
          child: Row(
            children: [
              Container(
                width: 40,
                height: 40,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: widget.isPublicMode
                      ? tokens.warning.withValues(alpha: 0.16)
                      : tokens.surface3,
                  borderRadius: BorderRadius.circular(ApertureTheme.radiusSm),
                ),
                child: Icon(
                  widget.isPublicMode ? Icons.public : Icons.public_off,
                  color: widget.isPublicMode
                      ? tokens.warning
                      : tokens.textMuted,
                  size: 22,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l10n.publicMode,
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        color: widget.isPublicMode
                            ? tokens.warning
                            : tokens.text,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      widget.isApplying
                          ? l10n.publicModeApplying
                          : l10n.publicModeHintDesc,
                      style: Theme.of(
                        context,
                      ).textTheme.bodySmall?.copyWith(color: tokens.textMuted),
                    ),
                  ],
                ),
              ),
              if (widget.isApplying)
                const SizedBox(
                  width: 24,
                  height: 24,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              else
                Container(
                  width: 48,
                  height: 28,
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(999),
                    color: widget.isPublicMode
                        ? tokens.warning
                        : tokens.surface3,
                    border: Border.all(
                      color: widget.isPublicMode
                          ? tokens.warning
                          : tokens.border,
                    ),
                  ),
                  child: AnimatedAlign(
                    duration: const Duration(milliseconds: 200),
                    // Directional, so the thumb travels toward the end edge in
                    // Arabic exactly as it does in English.
                    alignment: widget.isPublicMode
                        ? AlignmentDirectional.centerEnd
                        : AlignmentDirectional.centerStart,
                    child: Container(
                      width: 22,
                      height: 22,
                      margin: const EdgeInsets.all(2),
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        color: widget.isPublicMode
                            ? tokens.accentInk
                            : tokens.textFaint,
                      ),
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

// Theme button widget with proper focus handling
class ThemeButton extends StatefulWidget {
  final FocusNode focusNode;
  final AppThemeMode mode;
  final ThemeData theme;
  final bool isSelected;
  final VoidCallback onSelect;
  final VoidCallback onArrowRight;
  final VoidCallback onArrowLeft;
  final VoidCallback onArrowUp;

  const ThemeButton({
    super.key,
    required this.focusNode,
    required this.mode,
    required this.theme,
    required this.isSelected,
    required this.onSelect,
    required this.onArrowRight,
    required this.onArrowLeft,
    required this.onArrowUp,
  });

  @override
  State<ThemeButton> createState() => _ThemeButtonState();
}

class _ThemeButtonState extends State<ThemeButton> {
  bool _isFocused = false;

  @override
  void initState() {
    super.initState();
    widget.focusNode.addListener(_onFocusChange);
  }

  @override
  void dispose() {
    widget.focusNode.removeListener(_onFocusChange);
    super.dispose();
  }

  void _onFocusChange() {
    if (mounted) {
      setState(() {
        _isFocused = widget.focusNode.hasFocus;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final tokens = context.aperture;
    return Focus(
      focusNode: widget.focusNode,
      canRequestFocus: true,
      descendantsAreFocusable: false,
      onKeyEvent: (node, event) {
        if (event is KeyDownEvent) {
          if (event.logicalKey == LogicalKeyboardKey.arrowRight) {
            widget.onArrowRight();
            return KeyEventResult.handled;
          } else if (event.logicalKey == LogicalKeyboardKey.arrowLeft) {
            widget.onArrowLeft();
            return KeyEventResult.handled;
          } else if (event.logicalKey == LogicalKeyboardKey.arrowUp) {
            widget.onArrowUp();
            return KeyEventResult.handled;
          } else if (event.logicalKey == LogicalKeyboardKey.select ||
              event.logicalKey == LogicalKeyboardKey.enter) {
            widget.onSelect();
            return KeyEventResult.handled;
          }
        }
        return KeyEventResult.ignored;
      },
      child: GestureDetector(
        onTap: widget.onSelect,
        child: MouseRegion(
          cursor: SystemMouseCursors.click,
          // Selection is the bracket and a soft fill; focus is the ring.
          // Both can be true at once and stay distinguishable.
          child: ApertureBracket(
            radius: ApertureTheme.radiusSm,
            fill: widget.isSelected ? tokens.accentSoft : tokens.surface1,
            bracket: widget.isSelected ? tokens.accent : null,
            borderColor: _isFocused ? tokens.accent : tokens.border,
            borderWidth: _isFocused ? 2 : 1,
            padding: const EdgeInsetsDirectional.only(
              start: 14,
              end: 14,
              top: 12,
              bottom: 12,
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                // The mode's own accent, previewed as a dot.
                Container(
                  width: 12,
                  height: 12,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: widget.theme.colorScheme.primary,
                  ),
                ),
                const SizedBox(width: 10),
                Text(
                  widget.mode.name.toUpperCase(),
                  style: Theme.of(context).textTheme.labelMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                    letterSpacing: 0.7,
                    color: widget.isSelected ? tokens.accent : tokens.text,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

// ThemeModeSelector: isolates theme buttons so only this subtree rebuilds
class ThemeModeSelector extends StatelessWidget {
  final List<FocusNode> themeFocusNodes;
  final FocusNode previousFocusNode;

  const ThemeModeSelector({
    super.key,
    required this.themeFocusNodes,
    required this.previousFocusNode,
  });

  @override
  Widget build(BuildContext context) {
    final current = context.select<ServiceManager, AppThemeMode>(
      (s) => s.currentThemeMode,
    );

    return FocusTraversalGroup(
      child: Wrap(
        spacing: 12,
        runSpacing: 12,
        children: AppThemeMode.values.asMap().entries.map((entry) {
          final index = entry.key;
          final mode = entry.value;
          final isSelected = current == mode;
          final theme = AppTheme.getTheme(mode);
          final focusNode = themeFocusNodes[index];

          return ThemeButton(
            focusNode: focusNode,
            mode: mode,
            theme: theme,
            isSelected: isSelected,
            onSelect: () => context.read<ServiceManager>().setTheme(mode),
            onArrowRight: () {
              if (index < themeFocusNodes.length - 1) {
                themeFocusNodes[index + 1].requestFocus();
              } else {
                themeFocusNodes[0].requestFocus();
              }
            },
            onArrowLeft: () {
              if (index > 0) {
                themeFocusNodes[index - 1].requestFocus();
              } else {
                themeFocusNodes[themeFocusNodes.length - 1].requestFocus();
              }
            },
            onArrowUp: () => previousFocusNode.requestFocus(),
          );
        }).toList(),
      ),
    );
  }
}

/// PC-DEF-077. Offers the one explicit copy of an older shared workspace.
class _LegacyWorkspaceTile extends StatelessWidget {
  const _LegacyWorkspaceTile({
    required this.path,
    required this.busy,
    required this.onImport,
    required this.onHide,
  });

  final String path;
  final bool busy;
  final Future<void> Function() onImport;
  final Future<void> Function() onHide;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Card(
      margin: EdgeInsets.zero,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(
              Icons.drive_file_move_outline,
              size: 20,
              color: theme.colorScheme.primary,
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    l10n.legacyWorkspaceTitle,
                    style: theme.textTheme.titleSmall,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    // A path is left-to-right text; isolated, an RTL sentence
                    // cannot move its slashes.
                    l10n.legacyWorkspaceBody('\u2066$path\u2069'),
                    style: theme.textTheme.bodySmall,
                  ),
                  const SizedBox(height: 8),
                  Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: [
                      SizedBox(
                        height: 40,
                        child: OutlinedButton.icon(
                          onPressed: busy ? null : () => unawaited(onImport()),
                          icon: busy
                              ? const SizedBox(
                                  width: 16,
                                  height: 16,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Icon(Icons.content_copy, size: 16),
                          label: Text(l10n.legacyWorkspaceImport),
                        ),
                      ),
                      SizedBox(
                        height: 40,
                        child: TextButton(
                          onPressed: busy ? null : () => unawaited(onHide()),
                          child: Text(l10n.legacyWorkspaceHide),
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Notification permission state, and a way back for someone who said no.
///
/// PC-DEF-058. PocketClaw's foreground notification is how the owner sees that the
/// service is running, and on a fresh install it never appeared because
/// `POST_NOTIFICATIONS` was declared but never requested.
class _NotificationPermissionTile extends StatelessWidget {
  const _NotificationPermissionTile({
    required this.status,
    required this.onOpenSettings,
  });

  final NotificationPermissionStatus status;
  final Future<void> Function() onOpenSettings;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final ok = status.expectedVisible;

    return Card(
      margin: EdgeInsets.zero,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(
              ok ? Icons.notifications_active : Icons.notifications_off,
              size: 20,
              color: ok ? theme.colorScheme.primary : theme.colorScheme.error,
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    l10n.notificationPermissionTitle,
                    style: theme.textTheme.titleSmall,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    ok
                        ? l10n.notificationPermissionGranted
                        : l10n.notificationPermissionBlocked,
                    style: theme.textTheme.bodySmall,
                  ),
                  if (status.shouldOfferSettings) ...[
                    const SizedBox(height: 8),
                    // A full-height labelled control, not an icon: this screen is
                    // used on a phone.
                    SizedBox(
                      height: 40,
                      child: OutlinedButton.icon(
                        onPressed: () => unawaited(onOpenSettings()),
                        icon: const Icon(Icons.open_in_new, size: 16),
                        label: Text(l10n.notificationPermissionOpenSettings),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
