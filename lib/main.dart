import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:animations/animations.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:provider/provider.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/core/background_service.dart';
import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/core/app_identity.dart';
import 'package:pocketclaw/src/ui/dashboard_page.dart';
import 'package:pocketclaw/src/ui/config_page.dart';
import 'package:pocketclaw/src/ui/webview_page.dart';
import 'package:pocketclaw/src/ui/log_page.dart';
import 'package:tray_manager/tray_manager.dart';
import 'package:window_manager/window_manager.dart';
import 'package:windows_single_instance/windows_single_instance.dart';
import 'dart:io';
import 'package:pocketclaw/src/ui/widgets/adaptive_action_bar.dart';
import 'package:pocketclaw/src/ui/widgets/tv_focusable.dart';

void main(List<String> args) async {
  WidgetsFlutterBinding.ensureInitialized();

  // Single Instance Check (Windows only)
  if (Platform.isWindows) {
    await WindowsSingleInstance.ensureSingleInstance(
      args,
      "pocketclaw_instance_key",
      onSecondWindow: (newArgs) {
        windowManager.show();
        windowManager.focus();
      },
    );
  }

  // Initialize Window Manager (desktop platforms only)
  if (!Platform.isAndroid && !Platform.isIOS) {
    await windowManager.ensureInitialized();
    WindowOptions windowOptions = const WindowOptions(
      size: Size(1024, 768),
      minimumSize: Size(850, 650),
      center: true,
      skipTaskbar: false,
      titleBarStyle: TitleBarStyle.normal,
    );
    windowManager.waitUntilReadyToShow(windowOptions, () async {
      await windowManager.show();
      await windowManager.focus();
      await windowManager.setPreventClose(true);
    });
  }

  if (Platform.isAndroid) {
    await initializeBackgroundService();
  }

  final service = ServiceManager();
  await service.init();

  runApp(ChangeNotifierProvider.value(value: service, child: const MainApp()));

  // The only auto-start boundary: a true app-process launch. Nothing on resume.
  unawaited(service.evaluateLaunchAutoStart());
}

class MainApp extends StatelessWidget {
  const MainApp({super.key});

  @override
  Widget build(BuildContext context) {
    final service = context.watch<ServiceManager>();

    return MaterialApp(
      debugShowCheckedModeBanner: false,
      title: AppIdentity.productName,
      // One design system emits both brightnesses. Light mode used to come
      // from PocketClawDesign and dark mode from an unrelated FlexColorScheme
      // theme, so toggling did not adjust PocketClaw, it swapped products.
      theme: ApertureTheme.light(service.currentThemeMode),
      darkTheme: ApertureTheme.dark(service.currentThemeMode),
      themeMode: ThemeMode.system,
      locale: service.currentLocale,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: AppLocalizations.supportedLocales,
      home: const MainShell(),
    );
  }
}

class MainShell extends StatefulWidget {
  const MainShell({super.key});

  @override
  State<MainShell> createState() => _MainShellState();
}

class _MainShellState extends State<MainShell>
    with TrayListener, WindowListener {
  int _selectedIndex = 0;
  ServiceManager? _serviceManager;
  bool _configIsDirty = false;
  Future<void> Function()? _saveFn;
  String _webPath = '';

  void _onConfigDirtyChanged(bool dirty) {
    setState(() => _configIsDirty = dirty);
  }

  void _onSaveFnReady(Future<void> Function()? fn) {
    _saveFn = fn;
  }

  /// Opens a Dashboard route in the embedded console tab. Shared by every
  /// Settings card that hands work to the existing web UI.
  Future<void> _openConsolePath(String path) async {
    await _onNavTap(1, webPath: path);
  }

  String _webUrl(String baseUrl) {
    final base = Uri.parse(baseUrl);
    // Hand the app's language to the embedded Dashboard through i18next's own
    // query-string detector, which the console already configures. The console
    // ships fewer locales than the app, so an unsupported code simply falls
    // back to its English resources rather than failing.
    return base
        .replace(
          path: _webPath,
          queryParameters: {'lng': _consoleLanguageTag()},
        )
        .toString();
  }

  /// The locale tag handed to the embedded Dashboard.
  String _consoleLanguageTag() {
    final locale = context.read<ServiceManager>().currentLocale;
    final country = locale.countryCode;
    if (country != null && country.isNotEmpty) {
      return '${locale.languageCode}-$country';
    }
    return locale.languageCode;
  }

  Future<void> _onNavTap(int index, {String? webPath}) async {
    if (_selectedIndex == 3 && index != 3 && _configIsDirty) {
      final l10n = AppLocalizations.of(context)!;
      final result = await showDialog<bool>(
        context: context,
        builder: (ctx) => AlertDialog(
          title: Text(l10n.unsavedChanges),
          content: Text(l10n.unsavedChangesHint),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(false),
              child: Text(l10n.discard),
            ),
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(true),
              child: Text(l10n.save),
            ),
          ],
        ),
      );
      if (result == true) {
        if (!mounted) return;
        await _saveFn?.call();
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(AppLocalizations.of(context)!.saved)),
        );
      } else {
        _onConfigDirtyChanged(false);
      }
    }
    setState(() {
      _selectedIndex = index;
      if (index == 1) _webPath = webPath ?? '';
    });
  }

  bool get _supportsTray => !Platform.isAndroid && !Platform.isIOS;

  void _onServiceChanged() {
    if (!mounted) return;
    _initTray();
  }

  @override
  void initState() {
    super.initState();
    windowManager.addListener(this);
    if (_supportsTray) {
      trayManager.addListener(this);
    }
    // Defer tray init to didChangeDependencies where `context` and
    // inherited widgets (localizations) are available.
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_serviceManager == null) {
      _serviceManager = context.read<ServiceManager>();
      if (_supportsTray) {
        WidgetsBinding.instance.addPostFrameCallback((_) => _initTray());
        _serviceManager!.addListener(_onServiceChanged);
      }
    }
  }

  Future<void> _initTray() async {
    // Capture context-derived values before any await
    final l10n = AppLocalizations.of(context)!;
    final service = context.read<ServiceManager>();

    // Standardizing on the provided .ico for Windows tray and process
    try {
      await trayManager.setIcon(
        Platform.isWindows ? 'assets/icon.ico' : 'assets/app_icon.png',
      );
    } catch (e) {
      debugPrint('Tray icon error: $e');
    }

    Menu menu = Menu(
      items: [
        MenuItem(key: 'show_window', label: l10n.showWindow),
        MenuItem.separator(),
        MenuItem(
          key: 'start_service',
          label: l10n.run,
          disabled: service.status == ServiceStatus.running,
        ),
        MenuItem(
          key: 'stop_service',
          label: l10n.stop,
          disabled: service.status == ServiceStatus.stopped,
        ),
        MenuItem.separator(),
        MenuItem(key: 'exit_app', label: l10n.exit),
      ],
    );
    await trayManager.setContextMenu(menu);
    if (!Platform.isWindows) {
      trayManager.setTitle(l10n.appTitle);
    }
    if (!Platform.isLinux) {
      // Linux does not support tooltips, so we include the app name in the menu
      await trayManager.setToolTip(l10n.appTitle);
    }
  }

  @override
  void onTrayMenuItemClick(MenuItem menuItem) {
    final service = context.read<ServiceManager>();
    if (menuItem.key == 'show_window') {
      windowManager.show();
      windowManager.focus();
    } else if (menuItem.key == 'start_service') {
      service.start();
    } else if (menuItem.key == 'stop_service') {
      service.stop();
    } else if (menuItem.key == 'exit_app') {
      service.stop();
      exit(0);
    }
  }

  @override
  Widget build(BuildContext context) {
    // Re-init tray when status changes to update menu (disabled states)
    context.watch<ServiceManager>();

    // was previously used to decide rail vs bottom bar; handled by AdaptiveActionBar
    // Build actions that mirror previous NavigationRail / NavigationBar
    final actions = <Widget>[
      _buildNavButton(
        index: 0,
        tooltip: 'Status',
        icon: Icons.dashboard_outlined,
        selectedIcon: Icons.dashboard,
      ),
      _buildNavButton(
        index: 1,
        tooltip: 'Web',
        icon: Icons.language_outlined,
        selectedIcon: Icons.language,
      ),
      _buildNavButton(
        index: 2,
        tooltip: 'Logs',
        icon: Icons.article_outlined,
        selectedIcon: Icons.article,
      ),
      _buildNavButton(
        index: 3,
        tooltip: 'Settings',
        icon: Icons.settings_outlined,
        selectedIcon: Icons.settings,
      ),
    ];

    return Scaffold(
      body: AdaptiveActionBar(
        content: PageTransitionSwitcher(
          transitionBuilder: (child, primaryAnimation, secondaryAnimation) {
            return SharedAxisTransition(
              animation: primaryAnimation,
              secondaryAnimation: secondaryAnimation,
              transitionType: SharedAxisTransitionType.vertical,
              child: child,
            );
          },
          child: IndexedStack(
            key: ValueKey<int>(_selectedIndex),
            index: _selectedIndex,
            children: [
              const DashboardPage(),
              Consumer<ServiceManager>(
                builder: (context, service, _) => WebViewPage(
                  key: ValueKey<String>(_webPath),
                  url: _webUrl(service.webUrl),
                  onGoToDashboard: () => _onNavTap(0),
                ),
              ),
              const LogPage(),
              ConfigPage(
                onDirtyChanged: _onConfigDirtyChanged,
                onSaveFnReady: _onSaveFnReady,
                onManageTelegram: _openConsolePath,
                onManageConsole: _openConsolePath,
              ),
            ],
          ),
        ),
        actions: actions,
      ),
    );
  }

  /// Build navigation button with clear focus and selected state indicators.
  ///
  /// Selection and focus stay two different shapes and must never be merged:
  /// a control can be focused and not selected, selected and not focused, or
  /// both, and all four have to be distinguishable. On a TV the focus ring is
  /// the cursor, so [TVFocusable] keeps drawing it exactly as before; Aperture
  /// only changes what *selected* looks like.
  Widget _buildNavButton({
    required int index,
    required String tooltip,
    required IconData icon,
    required IconData selectedIcon,
  }) {
    final isSelected = _selectedIndex == index;
    final tokens = context.aperture;

    return TVFocusable(
      autofocus: index == 0,
      onTap: () => _onNavTap(index),
      borderRadius: BorderRadius.circular(ApertureTheme.radiusSm),
      focusBorderWidth: 2.5,
      focusBorderColor: tokens.accent,
      focusBackgroundColor: tokens.accentSoft,
      child: Container(
        width: 48,
        height: 48,
        decoration: BoxDecoration(
          color: isSelected ? tokens.accentSoft : Colors.transparent,
          borderRadius: BorderRadius.circular(ApertureTheme.radiusSm),
        ),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            // The bracket, rotated for a bottom bar. It is a separate strip
            // rather than a border side because Flutter refuses a
            // borderRadius on a border whose sides differ in colour.
            Container(
              width: 22,
              height: 2,
              margin: const EdgeInsets.only(bottom: 6),
              decoration: BoxDecoration(
                color: isSelected ? tokens.accent : Colors.transparent,
                borderRadius: BorderRadius.circular(1),
              ),
            ),
            Icon(
              isSelected ? selectedIcon : icon,
              color: isSelected ? tokens.accent : tokens.textFaint,
              size: 24,
            ),
          ],
        ),
      ),
    );
  }

  @override
  void onTrayIconMouseDown() {
    windowManager.show();
    windowManager.focus();
  }

  @override
  void onTrayIconRightMouseDown() {
    trayManager.popUpContextMenu();
  }

  @override
  void onWindowClose() async {
    bool isPreventClose = await windowManager.isPreventClose();
    if (isPreventClose) {
      await windowManager.hide();
    }
  }

  @override
  void dispose() {
    windowManager.removeListener(this);
    if (_supportsTray) {
      trayManager.removeListener(this);
    }
    if (_serviceManager != null) {
      try {
        _serviceManager!.removeListener(_onServiceChanged);
      } catch (_) {}
    }
    super.dispose();
  }
}
