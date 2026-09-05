import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:file_picker/file_picker.dart';
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

  final String appVersion;
  final String coreVersion;
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
  final ValueChanged<bool>? onDirtyChanged;

  /// Called once with the save function, so MainShell can call it later.
  final void Function(Future<void> Function()? saveFn)? onSaveFnReady;
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
    this.onDirtyChanged,
    this.onSaveFnReady,
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
  static ConfigPageState? _current;
  static ConfigPageState? get current => _current;
  final _hostController = TextEditingController();
  final _publicAddressController = TextEditingController();
  final _portController = TextEditingController();
  final _pathController = TextEditingController();
  final _argsController = TextEditingController();

  // Focus nodes for TV navigation
  final _whatsNewFocusNode = FocusNode();
  final _aboutFocusNode = FocusNode();
  final _contextMemoryFocusNode = FocusNode();
  final _publicModeFocusNode = FocusNode();
  final _hostFocusNode = FocusNode();
  final _portFocusNode = FocusNode();
  final _pathFocusNode = FocusNode();
  final _browseFocusNode = FocusNode();
  final _checkFocusNode = FocusNode();
  final _argsFocusNode = FocusNode();
  final _saveFocusNode = FocusNode();
  final _firebaseFocusNode = FocusNode();
  final List<FocusNode> _themeFocusNodes = [];
  bool _firebaseAllowed = false;

  /// The release the notes belong to, and whether the user has read them.
  String? _whatsNewVersion;
  bool _whatsNewUnseen = false;

  // Dirty tracking
  String _originalHost = '';
  String _originalPort = '';
  String _originalPath = '';
  String _originalArgs = '';
  bool _isDirty = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _current = this;
    widget.onSaveFnReady?.call(_saveConfig);

    // Initialize theme focus nodes
    _themeFocusNodes.addAll(
      List.generate(AppThemeMode.values.length, (_) => FocusNode()),
    );

    // Watch for changes to mark dirty
    _hostController.addListener(_markDirty);
    _portController.addListener(_markDirty);
    _pathController.addListener(_markDirty);
    _argsController.addListener(_markDirty);

    _loadConfig();
    _loadWhatsNewState();
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

  void _markDirty() {
    if (!_isDirty &&
        (_hostController.text != _originalHost ||
            _portController.text != _originalPort ||
            _pathController.text != _originalPath ||
            _argsController.text != _originalArgs)) {
      setState(() => _isDirty = true);
      widget.onDirtyChanged?.call(true);
    }
  }

  Future<void> _loadConfig() async {
    // 统一从 ServiceManager 加载配置，所有平台使用相同方式
    final service = context.read<ServiceManager>();
    final allowed = await service.isDeviceFeedbackAllowed();

    // 暂时移除监听器，避免设置 controller 值时触发 _markDirty
    _hostController.removeListener(_markDirty);
    _portController.removeListener(_markDirty);
    _pathController.removeListener(_markDirty);
    _argsController.removeListener(_markDirty);

    if (mounted) {
      setState(() {
        _hostController.text = service.host;
        _portController.text = service.port.toString();
        _pathController.text = service.binaryPath;
        _argsController.text = service.arguments;
        _firebaseAllowed = allowed;
        _originalHost = _hostController.text;
        _originalPort = _portController.text;
        _originalPath = _pathController.text;
        _originalArgs = _argsController.text;
        _isDirty = false;
      });
    }

    // 恢复监听器
    _hostController.addListener(_markDirty);
    _portController.addListener(_markDirty);
    _pathController.addListener(_markDirty);
    _argsController.addListener(_markDirty);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _current = null;
    _hostController.dispose();
    _publicAddressController.dispose();
    _portController.dispose();
    _pathController.dispose();
    _argsController.dispose();

    _whatsNewFocusNode.dispose();
    _aboutFocusNode.dispose();
    _contextMemoryFocusNode.dispose();
    _publicModeFocusNode.dispose();
    _hostFocusNode.dispose();
    _portFocusNode.dispose();
    _pathFocusNode.dispose();
    _browseFocusNode.dispose();
    _checkFocusNode.dispose();
    _argsFocusNode.dispose();
    _firebaseFocusNode.dispose();
    for (final node in _themeFocusNodes) {
      node.dispose();
    }

    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    super.didChangeAppLifecycleState(state);
    if (state == AppLifecycleState.resumed && Platform.isAndroid) {
      // App resumed from settings (e.g., storage permission granted)
      // Refresh workspace path to get the correct path after permission change
      context.read<ServiceManager>().refreshWorkspacePath();
    }
  }

  Future<void> _saveConfig() async {
    final service = context.read<ServiceManager>();
    final port = int.tryParse(_portController.text);
    final wasRunning = service.status == ServiceStatus.running;

    try {
      if (port != null) {
        final String? binaryArg = (Platform.isWindows || Platform.isAndroid)
            ? null
            : _pathController.text;

        await service.updateConfig(
          service.publicMode ? '0.0.0.0' : _hostController.text,
          port,
          binaryPath: binaryArg,
          arguments: _argsController.text,
          publicMode: service.publicMode,
        );
      }
    } catch (e) {
      debugPrint('[ConfigPage] save failed: $e');
    }

    // Restart service if it was running to apply new settings
    if (wasRunning) {
      await service.stop();
      await service.start();
    }

    // 无论保存成功还是失败，都重置 dirty 状态并通知父组件
    if (!mounted) return;
    setState(() {
      _originalHost = _hostController.text;
      _originalPort = _portController.text;
      _originalPath = _pathController.text;
      _originalArgs = _argsController.text;
      _isDirty = false;
    });
    widget.onDirtyChanged?.call(false);
  }

  Future<void> _pickFile() async {
    FilePickerResult? result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['exe', 'bat', 'sh'],
    );

    if (result != null) {
      _pathController.text = result.files.single.path ?? '';
      await _saveConfig();
    }
  }

  Future<void> _togglePublicMode(bool value) async {
    final service = context.read<ServiceManager>();

    final applied = await service.applyPublicMode(
      value,
      port: int.tryParse(_portController.text) ?? 18800,
      arguments: _argsController.text,
    );

    if (!mounted) return;
    if (!service.publicMode) {
      setState(() => _hostController.text = '127.0.0.1');
    }
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

  String _normalizeAboutVersion(String value, AppLocalizations l10n) {
    final normalized = value.trim();
    if (normalized.isEmpty || normalized.toLowerCase() == 'unknown') {
      return l10n.aboutVersionUnavailable;
    }
    return normalized;
  }

  Widget _buildAboutVersionRow(
    BuildContext context, {
    required String label,
    required String value,
  }) {
    final textTheme = Theme.of(context).textTheme;
    final tokens = context.aperture;
    return Padding(
      padding: const EdgeInsetsDirectional.only(top: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 148,
            child: Text(
              label,
              style: textTheme.bodyMedium?.copyWith(
                color: tokens.textMuted,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
          // A version, a commit and a build stamp are values a human compares
          // character by character, which is exactly what mono is for. They
          // stay left-to-right in Arabic: a reversed digest is a bug, not
          // localization.
          Expanded(
            child: Directionality(
              textDirection: TextDirection.ltr,
              child: Text(
                value,
                style: textTheme.bodyMedium?.copyWith(
                  fontFamily: 'monospace',
                  fontFamilyFallback: const ['Menlo', 'Consolas'],
                  color: tokens.text,
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _showAboutDialog() async {
    final l10n = AppLocalizations.of(context)!;
    final aboutInfoFuture = widget.aboutInfoLoader?.call() ?? _loadAboutInfo();

    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.about),
        content: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                _aboutProjectName,
                style: Theme.of(
                  ctx,
                ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 12),
              Text(l10n.aboutDescription),
              const SizedBox(height: 16),
              FutureBuilder<AboutInfo>(
                future: aboutInfoFuture,
                builder: (ctx, snapshot) {
                  if (!snapshot.hasData && !snapshot.hasError) {
                    return const Padding(
                      padding: EdgeInsets.symmetric(vertical: 8),
                      child: Center(child: CircularProgressIndicator()),
                    );
                  }

                  final info =
                      snapshot.data ??
                      AboutInfo(
                        appVersion: l10n.aboutVersionUnavailable,
                        coreVersion: l10n.aboutVersionUnavailable,
                      );
                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      _buildAboutVersionRow(
                        ctx,
                        label: l10n.aboutAppVersionLabel,
                        value: _normalizeAboutVersion(info.appVersion, l10n),
                      ),
                      _buildAboutVersionRow(
                        ctx,
                        label: l10n.aboutCoreVersionLabel,
                        value: _normalizeAboutVersion(info.coreVersion, l10n),
                      ),
                    ],
                  );
                },
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: Text(l10n.close),
          ),
        ],
      ),
    );

    if (mounted) {
      _aboutFocusNode.requestFocus();
    }
  }

  Future<void> _toggleFirebase(BuildContext context) async {
    final service = context.read<ServiceManager>();
    final newValue = !_firebaseAllowed;
    final l10n = AppLocalizations.of(context)!;

    debugPrint(
      '[ConfigPage] Toggling device feedback: newValue=$newValue (current=$_firebaseAllowed)',
    );

    if (newValue) {
      debugPrint('[ConfigPage] Enabling device feedback...');
      await service.setDeviceFeedbackUploadAllowed(true);
      setState(() {
        _firebaseAllowed = true;
      });
      debugPrint('[ConfigPage] Triggering background upload...');
      service.triggerDeviceFeedbackUploadInBackground();
    } else {
      debugPrint('[ConfigPage] Disabling device feedback...');
      await service.setDeviceFeedbackUploadAllowed(false);
      if (!context.mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(l10n.deviceReportingDisabled)));
    }

    if (!newValue) {
      setState(() {
        _firebaseAllowed = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    // Use scoped Selectors below to avoid whole-page rebuilds when ServiceManager changes.

    return PopScope(
      canPop: !_isDirty,
      onPopInvokedWithResult: (didPop, result) async {
        if (didPop) return;
        final shouldDiscard = await showDialog<bool>(
          context: context,
          builder: (ctx) {
            final colorScheme = Theme.of(ctx).colorScheme;
            final btnStyle = TextStyle(color: colorScheme.secondary);
            return AlertDialog(
              title: Text(
                AppLocalizations.of(ctx)!.unsavedChanges,
                style: TextStyle(
                  color: colorScheme.secondary,
                  fontSize: 20,
                  fontWeight: FontWeight.bold,
                ),
              ),
              content: Text(
                AppLocalizations.of(ctx)!.unsavedChangesHint,
                style: TextStyle(color: colorScheme.secondary),
              ),
              actions: [
                TextButton(
                  onPressed: () => Navigator.of(ctx).pop(false),
                  child: Text(
                    AppLocalizations.of(ctx)!.cancel,
                    style: btnStyle,
                  ),
                ),
                TextButton(
                  onPressed: () => Navigator.of(ctx).pop(true),
                  child: Text(
                    AppLocalizations.of(ctx)!.discard,
                    style: btnStyle,
                  ),
                ),
              ],
            );
          },
        );
        if (shouldDiscard == true && context.mounted) {
          Navigator.of(context).pop();
        }
      },
      child: FocusTraversalGroup(
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
                selector: (_, s) => (
                  isPublic: s.publicMode,
                  isApplying: s.isApplyingPublicMode,
                ),
                builder: (_, publicModeState, _) => PublicModeToggle(
                  focusNode: _publicModeFocusNode,
                  isPublicMode: publicModeState.isPublic,
                  isApplying: publicModeState.isApplying,
                  onToggle: _togglePublicModeFromFocus,
                  onArrowDown: () => _hostFocusNode.requestFocus(),
                  onArrowUp: () => _aboutFocusNode.requestFocus(),
                ),
              ),
              const SizedBox(height: 16),

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

              Text(
                l10n.address,
                style: Theme.of(context).textTheme.labelLarge?.copyWith(
                  color: context.aperture.textMuted,
                ),
              ),
              const SizedBox(height: 8),

              // Host text field - only depends on `publicMode`
              Selector<ServiceManager, ({bool isPublic, String? url})>(
                selector: (_, s) =>
                    (isPublic: s.publicMode, url: s.publicDashboardUrl),
                builder: (_, addressState, _) {
                  final isPublicMode = addressState.isPublic;
                  _publicAddressController.text =
                      addressState.url ?? l10n.unableToGetDeviceIp;
                  return FocusableTextField(
                    controller: isPublicMode
                        ? _publicAddressController
                        : _hostController,
                    focusNode: _hostFocusNode,
                    label: l10n.address,
                    enabled: !isPublicMode,
                    nextFocusNode: _portFocusNode,
                    prevFocusNode: _publicModeFocusNode,
                  );
                },
              ),
              const SizedBox(height: 16),

              Text(
                l10n.port,
                style: Theme.of(context).textTheme.labelLarge?.copyWith(
                  color: context.aperture.textMuted,
                ),
              ),
              const SizedBox(height: 8),

              // Port text field
              FocusableTextField(
                controller: _portController,
                focusNode: _portFocusNode,
                label: l10n.port,
                keyboardType: TextInputType.number,
                nextFocusNode:
                    (!Platform.isWindows &&
                        !Platform.isAndroid &&
                        !Platform.isMacOS &&
                        !Platform.isLinux)
                    ? _pathFocusNode
                    : _argsFocusNode,
                prevFocusNode: _hostFocusNode,
              ),
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

              if (!Platform.isWindows &&
                  !Platform.isAndroid &&
                  !Platform.isMacOS &&
                  !Platform.isLinux)
                Row(
                  children: [
                    Expanded(
                      child: FocusableTextField(
                        controller: _pathController,
                        focusNode: _pathFocusNode,
                        label: l10n.binaryPath,
                        nextFocusNode: _browseFocusNode,
                        prevFocusNode: _portFocusNode,
                      ),
                    ),
                    const SizedBox(width: 8),
                    SizedBox(
                      width: 100,
                      child: Column(
                        children: [
                          Builder(
                            builder: (ctx) {
                              final cs = Theme.of(ctx).colorScheme;
                              return FocusableButton(
                                focusNode: _browseFocusNode,
                                onPressed: _pickFile,
                                nextFocusNode: _checkFocusNode,
                                prevFocusNode: _pathFocusNode,
                                style: ElevatedButton.styleFrom(
                                  backgroundColor: cs.primary,
                                  foregroundColor: cs.onPrimary,
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 12,
                                    vertical: 14,
                                  ),
                                  minimumSize: const Size(100, 48),
                                  shape: RoundedRectangleBorder(
                                    borderRadius: BorderRadius.circular(8),
                                  ),
                                  elevation: 2,
                                ),
                                child: Row(
                                  mainAxisAlignment: MainAxisAlignment.center,
                                  children: [
                                    const Icon(Icons.folder_open, size: 20),
                                    const SizedBox(width: 4),
                                    Text(l10n.browse),
                                  ],
                                ),
                              );
                            },
                          ),
                          const SizedBox(height: 8),
                          Builder(
                            builder: (ctx) {
                              final messenger = ScaffoldMessenger.of(ctx);
                              final local = AppLocalizations.of(ctx)!;
                              final cs = Theme.of(ctx).colorScheme;
                              final service = ctx.read<ServiceManager>();
                              return FocusableButton(
                                focusNode: _checkFocusNode,
                                onPressed: () async {
                                  final code = await service.validateBinary(
                                    _pathController.text,
                                  );
                                  String msg;
                                  if (code) {
                                    msg = local.coreValid;
                                  } else {
                                    final ec = service.lastErrorCode;
                                    if (ec == 'core.binary_missing') {
                                      msg = local.coreBinaryMissing;
                                    } else if (ec == 'core.invalid_binary') {
                                      msg = local.coreInvalidBinary;
                                    } else if (ec == 'core.start_failed') {
                                      msg = local.coreStartFailed;
                                    } else {
                                      msg = local.coreUnknownError(ec ?? '');
                                    }
                                  }
                                  messenger.showSnackBar(
                                    SnackBar(content: Text(msg)),
                                  );
                                },
                                nextFocusNode: _argsFocusNode,
                                prevFocusNode: _browseFocusNode,
                                style: ElevatedButton.styleFrom(
                                  backgroundColor: cs.secondary,
                                  foregroundColor: cs.onSecondary,
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 12,
                                    vertical: 14,
                                  ),
                                  minimumSize: const Size(100, 48),
                                  shape: RoundedRectangleBorder(
                                    borderRadius: BorderRadius.circular(8),
                                  ),
                                  elevation: 2,
                                ),
                                child: Row(
                                  mainAxisAlignment: MainAxisAlignment.center,
                                  children: [
                                    const Icon(
                                      Icons.check_circle_outline,
                                      size: 20,
                                    ),
                                    const SizedBox(width: 4),
                                    Text(local.check),
                                  ],
                                ),
                              );
                            },
                          ),
                        ],
                      ),
                    ),
                  ],
                )
              else
                const SizedBox.shrink(),
              const SizedBox(height: 16),

              Text(
                l10n.arguments,
                style: Theme.of(context).textTheme.titleMedium?.copyWith(
                  color: Theme.of(context).colorScheme.onSurface.withAlpha(153),
                ),
              ),
              const SizedBox(height: 8),

              // Arguments text field
              FocusableTextField(
                controller: _argsController,
                focusNode: _argsFocusNode,
                label: l10n.arguments,
                hint: l10n.argumentsHint,
                nextFocusNode: _saveFocusNode,
                prevFocusNode: (!Platform.isWindows && !Platform.isAndroid)
                    ? _checkFocusNode
                    : _portFocusNode,
              ),
              if (Platform.isAndroid) ...[
                const SizedBox(height: 16),
                Text(
                  l10n.workspaceDirectory,
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    color: Theme.of(
                      context,
                    ).colorScheme.onSurface.withAlpha(153),
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
                      color: Theme.of(
                        context,
                      ).colorScheme.outline.withAlpha(60),
                    ),
                  ),
                  child: Selector<ServiceManager, String>(
                    selector: (_, s) => s.workspacePath,
                    builder: (_, path, _) => Text(
                      path,
                      style: Theme.of(context).textTheme.bodyMedium,
                    ),
                  ),
                ),
              ],
              const SizedBox(height: 24),

              Selector<ServiceManager, bool>(
                selector: (_, s) => s.isDeviceFeedbackEnabled,
                builder: (_, enabled, _) {
                  if (!enabled) return const SizedBox.shrink();
                  return Selector<ServiceManager, String?>(
                    selector: (_, s) => s.lastDeviceFeedbackSyncMessage,
                    builder: (_, msg, _) => DeviceFeedbackToggle(
                      focusNode: _firebaseFocusNode,
                      isAllowed: _firebaseAllowed,
                      statusMessage: msg,
                      onToggle: () => _toggleFirebase(context),
                      onArrowDown: () => _saveFocusNode.requestFocus(),
                      onArrowUp: () => _saveFocusNode.requestFocus(),
                    ),
                  );
                },
              ),

              const SizedBox(height: 24),

              // Save button
              Builder(
                builder: (ctx) {
                  final cs = Theme.of(ctx).colorScheme;
                  final messenger = ScaffoldMessenger.of(ctx);
                  final local = AppLocalizations.of(ctx)!;
                  return FocusableButton(
                    focusNode: _saveFocusNode,
                    onPressed: () async {
                      await _saveConfig();
                      if (!ctx.mounted) return;
                      messenger.showSnackBar(
                        SnackBar(content: Text(local.saved)),
                      );
                    },
                    nextFocusNode: _themeFocusNodes.isNotEmpty
                        ? _themeFocusNodes.first
                        : _saveFocusNode,
                    prevFocusNode: _argsFocusNode,
                    style: ElevatedButton.styleFrom(
                      backgroundColor: cs.secondary,
                      foregroundColor: cs.onSecondary,
                      padding: const EdgeInsets.symmetric(
                        horizontal: 20,
                        vertical: 16,
                      ),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(10),
                      ),
                      elevation: 2,
                    ),
                    child: Text(l10n.save),
                  );
                },
              ),

              const SizedBox(height: 16),
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
                    focusNode: _saveFocusNode,
                    onPressed: () {},
                    prevFocusNode: _argsFocusNode,
                    nextFocusNode: _themeFocusNodes.isNotEmpty
                        ? _themeFocusNodes.first
                        : _saveFocusNode,
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
                              child: Text(
                                _getLanguageName(locale.languageCode),
                              ),
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
                saveFocusNode: _saveFocusNode,
              ),
              const SizedBox(height: 32),
            ],
          ),
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

class DeviceFeedbackToggle extends StatefulWidget {
  final FocusNode focusNode;
  final bool isAllowed;
  final String? statusMessage;
  final VoidCallback onToggle;
  final VoidCallback onArrowDown;
  final VoidCallback onArrowUp;

  const DeviceFeedbackToggle({
    super.key,
    required this.focusNode,
    required this.isAllowed,
    this.statusMessage,
    required this.onToggle,
    required this.onArrowDown,
    required this.onArrowUp,
  });

  @override
  State<DeviceFeedbackToggle> createState() => _DeviceFeedbackToggleState();
}

// ThemeModeSelector: isolates theme buttons so only this subtree rebuilds
class ThemeModeSelector extends StatelessWidget {
  final List<FocusNode> themeFocusNodes;
  final FocusNode saveFocusNode;

  const ThemeModeSelector({
    super.key,
    required this.themeFocusNodes,
    required this.saveFocusNode,
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
            onArrowUp: () => saveFocusNode.requestFocus(),
          );
        }).toList(),
      ),
    );
  }
}

class _DeviceFeedbackToggleState extends State<DeviceFeedbackToggle> {
  bool _isFocused = false;
  bool _hasUserToggled = false;

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

  void _handleToggle() {
    if (!_hasUserToggled) {
      setState(() {
        _hasUserToggled = true;
      });
    }
    widget.onToggle();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
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
            _handleToggle();
            return KeyEventResult.handled;
          }
        }
        return KeyEventResult.ignored;
      },
      child: GestureDetector(
        onTap: _handleToggle,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 200),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: _isFocused
                ? Theme.of(context).colorScheme.secondary.withAlpha(40)
                : null,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: _isFocused
                  ? Theme.of(context).colorScheme.secondary
                  : Theme.of(context).dividerColor,
              width: _isFocused ? 2 : 1,
            ),
            boxShadow: _isFocused
                ? [
                    BoxShadow(
                      color: Theme.of(
                        context,
                      ).colorScheme.secondary.withAlpha(40),
                      blurRadius: 8,
                      spreadRadius: 2,
                    ),
                  ]
                : null,
          ),
          child: Row(
            children: [
              AnimatedContainer(
                duration: const Duration(milliseconds: 200),
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: _isFocused
                      ? Theme.of(context).colorScheme.secondary.withAlpha(40)
                      : null,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Icon(
                  widget.isAllowed ? Icons.analytics : Icons.analytics_outlined,
                  color: _isFocused
                      ? Theme.of(context).colorScheme.secondary
                      : Theme.of(context).colorScheme.onSurface.withAlpha(150),
                  size: 24,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      l10n.deviceReportingTitle,
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        color: _isFocused
                            ? Theme.of(context).colorScheme.secondary
                            : null,
                        fontWeight: _isFocused
                            ? FontWeight.bold
                            : FontWeight.normal,
                      ),
                    ),
                    Text(
                      l10n.deviceReportingSubtitle,
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                    if (widget.statusMessage != null &&
                        widget.statusMessage!.isNotEmpty) ...[
                      const SizedBox(height: 4),
                      Text(
                        widget.statusMessage!,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                          color: Theme.of(context).colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ],
                ),
              ),
              Container(
                width: 48,
                height: 28,
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(14),
                  color: widget.isAllowed
                      ? Theme.of(context).colorScheme.secondary
                      : Theme.of(context).colorScheme.secondary.withAlpha(100),
                ),
                child: AnimatedAlign(
                  duration: Duration(milliseconds: _hasUserToggled ? 200 : 0),
                  alignment: widget.isAllowed
                      ? Alignment.centerRight
                      : Alignment.centerLeft,
                  child: Container(
                    width: 24,
                    height: 24,
                    margin: const EdgeInsets.all(2),
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: Theme.of(context).colorScheme.surface,
                      boxShadow: [
                        BoxShadow(
                          color: Colors.black.withAlpha(30),
                          blurRadius: 2,
                          offset: const Offset(0, 1),
                        ),
                      ],
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
