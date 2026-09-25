import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:animations/animations.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:provider/provider.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/core/app_identity.dart';
import 'package:pocketclaw/src/ui/dashboard_page.dart';
import 'package:pocketclaw/src/ui/config_page.dart';
import 'package:pocketclaw/src/ui/webview_page.dart';
import 'package:pocketclaw/src/ui/log_page.dart';
import 'package:pocketclaw/src/ui/widgets/adaptive_action_bar.dart';
import 'package:pocketclaw/src/ui/widgets/tv_focusable.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

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

class _MainShellState extends State<MainShell> {
  int _selectedIndex = 0;
  String _webPath = '';

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
    setState(() {
      _selectedIndex = index;
      if (index == 1) _webPath = webPath ?? '';
    });
  }

  @override
  Widget build(BuildContext context) {
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
              // Detail is requested only while Status is the selected tab.
              // The pages live in an IndexedStack and are never disposed on a
              // tab change, so the flag — not the widget lifecycle — is what
              // stops the extra work.
              DashboardPage(detailEnabled: _selectedIndex == 0),
              Consumer<ServiceManager>(
                builder: (context, service, _) => WebViewPage(
                  key: ValueKey<String>(_webPath),
                  url: _webUrl(service.webUrl),
                  onGoToDashboard: () => _onNavTap(0),
                ),
              ),
              const LogPage(),
              ConfigPage(
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
}
