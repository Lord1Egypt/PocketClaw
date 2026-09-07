import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:remixicon/remixicon.dart';
import 'package:pocketclaw/src/ui/widgets/tv_focusable.dart';
import 'package:pocketclaw/src/ui/status_sections.dart';

class DashboardPage extends StatefulWidget {
  const DashboardPage({super.key, this.detailEnabled = true});

  /// Whether this page is the selected tab. The detailed Status payload is
  /// only requested while it is, so an unwatched Status tab leaves the shared
  /// health poll costing exactly what it always did.
  final bool detailEnabled;

  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  // Held so dispose can turn detail off without looking up an ancestor on an
  // already-deactivated element, which is not allowed at that point.
  ServiceManager? _service;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _service = context.read<ServiceManager>();
    _syncDetailRequest();
  }

  @override
  void didUpdateWidget(covariant DashboardPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.detailEnabled != widget.detailEnabled) {
      _syncDetailRequest();
    }
  }

  @override
  void dispose() {
    // Leaving Status stops the detailed request. The page is kept alive by an
    // IndexedStack, so dispose alone would not be enough — detailEnabled is
    // what actually turns it off when the tab changes — but stopping here too
    // means no path can leave detail running with nothing to display it.
    _service?.setStatusDetailWanted(false);
    super.dispose();
  }

  void _syncDetailRequest() {
    final service = _service;
    if (service == null) return;
    final enabled = widget.detailEnabled;
    // Deferred past the current build: setStatusDetailWanted notifies
    // listeners, and notifying while this page is building would rebuild it
    // mid-frame.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (enabled && !mounted) return;
      service.setStatusDetailWanted(enabled);
    });
  }

  @override
  Widget build(BuildContext context) {
    final service = context.watch<ServiceManager>();
    final l10n = AppLocalizations.of(context)!;
    final colorScheme = Theme.of(context).colorScheme;
    final tokens = context.aperture;
    final isRunning = service.status == ServiceStatus.running;

    final connectableUrl = service.connectableDashboardUrl;

    return Scaffold(
      backgroundColor:
          Colors.transparent, // Let themed scaffoldBackground show through
      body: CustomScrollView(
        physics: const BouncingScrollPhysics(),
        slivers: [
          SliverAppBar(
            floating: true,
            backgroundColor: Colors.transparent,
            elevation: 0,
            centerTitle: false,
            title: Text(
              l10n.statusTitle,
              style: GoogleFonts.inter(
                fontWeight: FontWeight.w600,
                fontSize: 22,
                letterSpacing: -0.3,
                color: tokens.text,
              ),
            ),
            actions: [
              _buildStatusIndicator(context, service.status),
              const SizedBox(width: ApertureTheme.spaceMd),
            ],
          ),
          SliverPadding(
            padding: const EdgeInsets.symmetric(
              horizontal: ApertureTheme.spaceMd,
              vertical: ApertureTheme.spaceMd,
            ),
            sliver: SliverList(
              delegate: SliverChildListDelegate([
                // High-Impact Control Center
                Padding(
                  padding: const EdgeInsets.only(bottom: ApertureTheme.spaceLg),
                  child: Row(
                    children: [
                      Expanded(
                        child: TVFocusable(
                          onTap: isRunning ? service.stop : service.start,
                          borderRadius: BorderRadius.circular(
                            ApertureTheme.radiusMd,
                          ),
                          focusBorderColor: isRunning
                              ? tokens.danger
                              : tokens.accent,
                          child: Container(
                            padding: const EdgeInsets.symmetric(
                              vertical: 20,
                              horizontal: 24,
                            ),
                            decoration: BoxDecoration(
                              // Start is the primary action, so it is filled
                              // with the accent. Stop is destructive, so it
                              // wears Danger as an outline until it is used —
                              // an outlined destructive control is harder to
                              // hit by accident than a filled one.
                              color: isRunning
                                  ? tokens.dangerSoft
                                  : tokens.accent,
                              borderRadius: BorderRadius.circular(
                                ApertureTheme.radiusMd,
                              ),
                              border: Border.all(
                                color: isRunning
                                    ? tokens.danger
                                    : tokens.accent,
                                width: isRunning ? 1.5 : 1,
                              ),
                            ),
                            child: Row(
                              mainAxisAlignment: MainAxisAlignment.center,
                              children: [
                                Icon(
                                  isRunning
                                      ? Remix.stop_circle_fill
                                      : Remix.play_circle_fill,
                                  size: 28,
                                  color: isRunning
                                      ? tokens.danger
                                      : tokens.accentInk,
                                ),
                                const SizedBox(width: 14),
                                Flexible(
                                  child: Text(
                                    isRunning
                                        ? l10n.stopService
                                        : l10n.launchService,
                                    style: GoogleFonts.inter(
                                      fontSize: 17,
                                      fontWeight: FontWeight.w600,
                                      letterSpacing: 0.2,
                                      color: isRunning
                                          ? tokens.danger
                                          : tokens.accentInk,
                                    ),
                                    overflow: TextOverflow.ellipsis,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                // Glassmorphism Status Card
                LayoutBuilder(
                  builder: (context, constraints) {
                    // 窄屏时垂直布局，宽屏时水平布局
                    final isNarrow = constraints.maxWidth < 600;

                    Widget infoSection = _buildInfoSection(
                      context,
                      service,
                      colorScheme,
                      l10n,
                    );

                    Widget qrSection = _buildQrSection(
                      context,
                      connectableUrl,
                      colorScheme,
                    );

                    return Container(
                      decoration: BoxDecoration(
                        color: tokens.surface1,
                        borderRadius: BorderRadius.circular(
                          ApertureTheme.radiusMd,
                        ),
                        border: Border.all(color: tokens.border),
                      ),
                      clipBehavior: Clip.antiAlias,
                      // Endpoint, QR and the steps for using them are one
                      // access section. The steps used to sit at the very
                      // bottom of the page, below the Status metrics, where
                      // they read as a footnote about nothing in particular
                      // rather than as instructions for the QR directly above
                      // them.
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          if (isNarrow)
                            Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                infoSection,
                                Container(
                                  width: double.infinity,
                                  height: 1,
                                  color: tokens.border,
                                ),
                                Center(child: qrSection),
                              ],
                            )
                          else
                            Row(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Expanded(flex: 3, child: infoSection),
                                Container(
                                  width: 1,
                                  height: 240,
                                  color: tokens.border,
                                ),
                                qrSection,
                              ],
                            ),
                          Container(
                            width: double.infinity,
                            height: 1,
                            color: tokens.border,
                          ),
                          _buildAccessHint(context, service, l10n),
                        ],
                      ),
                    );
                  },
                ),
                const SizedBox(height: ApertureTheme.spaceLg),
                StatusSections(
                  gatewayRunning: isRunning,
                  appVersion: service.appVersion,
                  coreVersion: service.coreVersionLabel,
                  snapshot: service.statusSnapshot,
                ),
                const SizedBox(height: 100),
              ]),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildInfoSection(
    BuildContext context,
    ServiceManager service,
    ColorScheme colorScheme,
    AppLocalizations l10n,
  ) {
    final tokens = context.aperture;
    return Padding(
      padding: const EdgeInsets.all(ApertureTheme.spaceLg),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(
            children: [
              Container(
                width: 32,
                height: 32,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: tokens.surface3,
                  borderRadius: BorderRadius.circular(ApertureTheme.radiusXs),
                ),
                child: Icon(Remix.link_m, color: tokens.accent, size: 16),
              ),
              const SizedBox(width: 10),
              Flexible(
                child: Text(
                  l10n.endpoint,
                  style: GoogleFonts.inter(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    letterSpacing: 0.9,
                    color: tokens.textFaint,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          // 公共模式状态指示器
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                service.publicMode ? Icons.public : Icons.lock_outline,
                size: 15,
                color: service.publicMode ? tokens.warning : tokens.textMuted,
              ),
              const SizedBox(width: 6),
              Flexible(
                child: Text(
                  service.publicMode ? l10n.publicModeEnabled : l10n.localMode,
                  style: TextStyle(
                    fontSize: 13,
                    color: service.publicMode
                        ? tokens.warning
                        : tokens.textMuted,
                    fontWeight: FontWeight.w600,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          // Endpoint 显示地址：公共模式开启时使用设备IP，否则使用内部地址
          Builder(
            builder: (context) {
              // 公共模式开启但无法获取IP时显示警告
              if (service.publicMode && service.publicDashboardUrl == null) {
                return TVFocusable(
                  onTap: null,
                  borderRadius: BorderRadius.circular(8),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(vertical: 4),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          l10n.unableToGetDeviceIp,
                          style: GoogleFonts.firaCode(
                            fontSize: 16,
                            color: tokens.warning,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ],
                    ),
                  ),
                );
              }
              final displayUrl = service.connectableDashboardUrl!;
              return TVFocusable(
                onTap: () => launchUrl(Uri.parse(displayUrl)),
                borderRadius: BorderRadius.circular(8),
                child: Padding(
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  // A URL is a machine value: it stays left-to-right even
                  // in Arabic, because a reversed host is a bug.
                  child: Directionality(
                    textDirection: TextDirection.ltr,
                    child: Text(
                      displayUrl,
                      style: GoogleFonts.firaCode(
                        fontSize: 18,
                        color: tokens.accent,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ),
              );
            },
          ),
          const SizedBox(height: 16),
          Text(
            l10n.webAdmin,
            style: TextStyle(color: tokens.textFaint, fontSize: 13),
          ),
        ],
      ),
    );
  }

  /// The steps for reaching PocketClaw, rendered inside the access card
  /// directly under the QR they describe.
  ///
  /// Which steps apply depends on Public Mode, which is why this reads the
  /// service rather than taking a fixed string.
  Widget _buildAccessHint(
    BuildContext context,
    ServiceManager service,
    AppLocalizations l10n,
  ) {
    final tokens = context.aperture;
    return Padding(
      padding: const EdgeInsets.all(ApertureTheme.spaceMd),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(Icons.lightbulb_outline, size: 18, color: tokens.textFaint),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              service.publicMode ? l10n.publicModeHint : l10n.localModeHint,
              style: TextStyle(
                fontSize: 14,
                color: tokens.textMuted,
                height: 1.5,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildQrSection(
    BuildContext context,
    String? qrData,
    ColorScheme colorScheme,
  ) {
    final tokens = context.aperture;
    return Container(
      width: 200,
      height: 240,
      color: tokens.surface2,
      child: Center(
        child: Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            // The quiet zone stays white in both themes: a tinted or dark
            // ground behind a QR is a scan failure, not a style choice.
            color: Colors.white,
            borderRadius: BorderRadius.circular(ApertureTheme.radiusSm),
          ),
          child: qrData == null
              ? const Icon(
                  Icons.wifi_off_rounded,
                  size: 64,
                  color: Color(0xFFBFC7CF),
                )
              : DashboardAccessQrCode(data: qrData),
        ),
      ),
    );
  }

  /// The status chip: a dot and a word, never a dot alone.
  ///
  /// Running is genuinely live machine state, which is the one thing Signal is
  /// reserved for. Starting is transitional, so it is Warning. Stopped is not
  /// an error and does not get a colour at all.
  Widget _buildStatusIndicator(BuildContext context, ServiceStatus status) {
    final tokens = context.aperture;
    final l10n = AppLocalizations.of(context)!;
    final (Color color, String label) = switch (status) {
      ServiceStatus.running => (tokens.signal, l10n.statusActive),
      ServiceStatus.starting => (tokens.warning, l10n.statusSyncing),
      ServiceStatus.stopped => (tokens.textFaint, l10n.statusIdle),
    };
    return Container(
      padding: const EdgeInsetsDirectional.only(
        start: 8,
        end: 10,
        top: 5,
        bottom: 5,
      ),
      decoration: BoxDecoration(
        color: tokens.surface1,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: tokens.border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(color: color, shape: BoxShape.circle),
          ),
          const SizedBox(width: 7),
          Text(
            label,
            style: GoogleFonts.inter(
              color: color,
              fontWeight: FontWeight.w600,
              fontSize: 11,
              letterSpacing: 0.5,
            ),
          ),
        ],
      ),
    );
  }
}

/// Named wrapper so the connect URL encoded by the Dashboard QR remains a
/// testable part of the UI contract when LAN addressing changes.
class DashboardAccessQrCode extends StatelessWidget {
  const DashboardAccessQrCode({super.key, required this.data});

  final String data;

  @override
  Widget build(BuildContext context) {
    return QrImageView(
      data: data,
      version: QrVersions.auto,
      size: 140.0,
      gapless: true,
    );
  }
}
