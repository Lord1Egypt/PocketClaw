// PocketClaw Web Console - Web-based chat and management interface
//
// Provides a web UI for chatting with PocketClaw over the realtime channel,
// with configuration management and gateway process control.
//
// Usage:
//
//	go build -o picoclaw-web ./web/backend/
//	./picoclaw-web [config.json]
//	./picoclaw-web -public config.json

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/androiddns"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/coresource"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/netbind"
	"github.com/sipeed/picoclaw/web/backend/api"
	"github.com/sipeed/picoclaw/web/backend/dashboardauth"
	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
	"github.com/sipeed/picoclaw/web/backend/middleware"
	"github.com/sipeed/picoclaw/web/backend/utils"
)

const (
	appName = "PocketClaw"

	panicFile = "launcher_panic.log"
	logFile   = "launcher.log"
)

var (
	appVersion = config.Version

	httpRuntime *launcherHTTPRuntime
	serverAddr  string
	// browserLaunchURL is opened by openBrowser() (auto-open + tray "open console").
	browserLaunchURL string
	apiHandler       *api.Handler

	noBrowser *bool
)

func shouldEnableLauncherFileLogging(enableConsole, debug bool) bool {
	return !enableConsole || debug
}

// gatewayAutoStartEnabled reads the host's Gateway auto-start preference from
// the environment. An absent or unparsable value keeps the historical
// always-on behaviour, so only an explicit, well-formed "false" disables it.
func gatewayAutoStartEnabled(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return true
	}
	return enabled
}

func shouldEnableLocalAutoLogin(noBrowser bool, probeHost string) bool {
	return !noBrowser && isLoopbackLaunchHost(probeHost)
}

func isLoopbackLaunchHost(host string) bool {
	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	host = strings.Trim(host, "[]")
	if i := strings.LastIndex(host, "%"); i >= 0 {
		host = host[:i]
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func launcherBrowserLaunchSuffix(
	needsSetup bool,
	localAutoLogin *middleware.LauncherDashboardLocalAutoLogin,
) string {
	if needsSetup {
		return middleware.LauncherDashboardSetupPath
	}
	if localAutoLogin != nil {
		return localAutoLogin.URLPath()
	}
	return ""
}

func resolveLauncherHostInput(flagHost string, explicitFlag bool, envHost string) (string, bool, error) {
	if explicitFlag {
		normalized, err := netbind.NormalizeHostInput(flagHost)
		if err != nil {
			return "", false, err
		}
		return normalized, true, nil
	}

	envHost = strings.TrimSpace(envHost)
	if envHost == "" {
		return "", false, nil
	}

	normalized, err := netbind.NormalizeHostInput(envHost)
	if err != nil {
		return "", false, err
	}
	return normalized, true, nil
}

// launcherExplicitFlags reports which listen flags the caller actually supplied.
//
// This is flag.Visit rather than a value comparison, and the difference is the
// whole point: -public=false and an omitted -public both leave the parsed value
// false, and they must not mean the same thing. Visit only reports flags that
// were Set, so an explicit false is distinguishable from silence — which is
// what lets a host own the decision. See resolveLauncherPublicMode.
func launcherExplicitFlags(fs *flag.FlagSet) (port, host, public bool) {
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "port":
			port = true
		case "host":
			host = true
		case "public":
			public = true
		}
	})
	return port, host, public
}

// resolveLauncherPublicMode decides whether the dashboard listener may leave
// loopback.
//
// A supplied flag is the authority for this process; the persisted launcher
// config is consulted only when no flag was supplied. That ordering is the
// fix for PC-DEF-020. The Android host stores the user's Public Mode choice
// natively and used to pass -public only when it was on, so "off" arrived as
// silence — and silence fell through to launcher-config.json's `public` field,
// which the dashboard's own Config page can set to true. A user who enabled LAN
// access, saved that page and then switched the native toggle off got a
// loopback listener for the life of that process and a wildcard one on the next
// start, with the toggle still reading OFF. The host now always passes the
// value, so this function never reaches the persisted field on Android.
//
// The persisted field remains the authority on desktop, where there is no
// native toggle and the Config page is how Public Mode is set at all.
func resolveLauncherPublicMode(flagPublic, flagPublicExplicit, configPublic bool) bool {
	if flagPublicExplicit {
		return flagPublic
	}
	return configPublic
}

// effectiveLauncherExposure narrows the desired Public Mode to what is safe to
// bind right now, and reports whether it narrowed anything.
//
// Desired and effective are kept separate on purpose. The user's preference is
// not edited and not forgotten: an unclaimed dashboard simply is not offered
// beyond loopback, because until a password exists the first client to reach
// POST /api/auth/setup would own the agent. See PC-DEF-039.
func effectiveLauncherExposure(desiredPublic, dashboardInitialized bool) (effective bool, narrowed bool) {
	if desiredPublic && !dashboardInitialized {
		return false, true
	}
	return desiredPublic, false
}

func openLauncherListeners(hostInput string, public bool, port string) (netbind.OpenResult, error) {
	defaultMode := netbind.DefaultLoopback
	if strings.TrimSpace(hostInput) == "" && public {
		defaultMode = netbind.DefaultAny
	}

	plan, err := netbind.BuildPlan(hostInput, defaultMode)
	if err != nil {
		return netbind.OpenResult{}, err
	}
	return netbind.OpenPlan(plan, port)
}

func appendUniqueHost(hosts []string, seen map[string]struct{}, host string) []string {
	host = strings.TrimSpace(host)
	if host == "" {
		return hosts
	}
	key := strings.ToLower(host)
	if _, ok := seen[key]; ok {
		return hosts
	}
	seen[key] = struct{}{}
	return append(hosts, host)
}

func hasWildcardBindHosts(bindHosts []string) bool {
	for _, bindHost := range bindHosts {
		if netbind.IsUnspecifiedHost(bindHost) {
			return true
		}
	}
	return false
}

func wildcardBindHostFamilies(bindHosts []string) (hasIPv4, hasIPv6 bool) {
	for _, bindHost := range bindHosts {
		host := strings.TrimSpace(bindHost)
		if host == "" {
			continue
		}

		if !netbind.IsUnspecifiedHost(host) {
			continue
		}

		ip := net.ParseIP(strings.Trim(host, "[]"))
		if ip == nil {
			continue
		}
		if ip.To4() != nil {
			hasIPv4 = true
			continue
		}
		hasIPv6 = true
	}

	return hasIPv4, hasIPv6
}

func wildcardAdvertiseIP(bindHosts []string, ipv4, ipv6 string) string {
	hasIPv4Wildcard, hasIPv6Wildcard := wildcardBindHostFamilies(bindHosts)
	v4 := strings.TrimSpace(ipv4)
	v6 := strings.TrimSpace(ipv6)

	switch {
	case hasIPv4Wildcard && hasIPv6Wildcard:
		if v6 != "" {
			return v6
		}
		return v4
	case hasIPv6Wildcard:
		return v6
	case hasIPv4Wildcard:
		return v4
	default:
		return ""
	}
}

func advertiseIPForWildcardBindHosts(bindHosts []string) string {
	return wildcardAdvertiseIP(bindHosts, utils.GetLocalIPv4(), utils.GetLocalIPv6())
}

func appendLauncherConsoleHostList(hosts []string, seen map[string]struct{}, values []string) []string {
	for _, value := range values {
		hosts = appendUniqueHost(hosts, seen, value)
	}
	return hosts
}

func shouldShowLocalhostConsoleEntry(hostInput string) bool {
	normalizedHostInput := strings.TrimSpace(hostInput)
	if normalizedHostInput == "" {
		return true
	}

	for token := range strings.SplitSeq(normalizedHostInput, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if token == "*" || strings.EqualFold(token, "localhost") {
			return true
		}

		ip := net.ParseIP(strings.Trim(token, "[]"))
		if ip == nil {
			continue
		}
		if ip4 := ip.To4(); ip4 != nil {
			if ip4.String() == "127.0.0.1" || ip4.String() == "0.0.0.0" {
				return true
			}
			continue
		}
		if ip.String() == "::1" || ip.String() == "::" {
			return true
		}
	}

	return false
}

func isConsoleDisplayGlobalIPv6(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.To4() != nil {
		return false
	}
	ip = ip.To16()
	if ip == nil {
		return false
	}
	return ip[0]&0xe0 == 0x20
}

func launcherConsoleHostsWithLocalAddrs(
	hostInput string,
	public bool,
	ipv4s []string,
	globalIPv6s []string,
) []string {
	hosts := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)

	if shouldShowLocalhostConsoleEntry(hostInput) {
		hosts = appendUniqueHost(hosts, seen, "localhost")
	}

	normalizedHostInput := strings.TrimSpace(hostInput)
	if normalizedHostInput == "" {
		if public {
			hosts = appendLauncherConsoleHostList(hosts, seen, globalIPv6s)
			hosts = appendLauncherConsoleHostList(hosts, seen, ipv4s)
		}
		return hosts
	}

	hasStar := false
	hasIPv4Any := false
	hasIPv6Any := false
	for _, token := range strings.Split(normalizedHostInput, ",") {
		switch strings.TrimSpace(token) {
		case "*":
			hasStar = true
		case "0.0.0.0":
			hasIPv4Any = true
		case "::":
			hasIPv6Any = true
		}
	}

	if hasStar {
		hosts = appendLauncherConsoleHostList(hosts, seen, globalIPv6s)
		hosts = appendLauncherConsoleHostList(hosts, seen, ipv4s)
		return hosts
	}

	for _, token := range strings.Split(normalizedHostInput, ",") {
		token = strings.TrimSpace(token)
		if token == "" || strings.EqualFold(token, "localhost") || netbind.IsLoopbackHost(token) {
			continue
		}

		ip := net.ParseIP(strings.Trim(token, "[]"))
		switch {
		case token == "::":
			hosts = appendLauncherConsoleHostList(hosts, seen, globalIPv6s)
		case token == "0.0.0.0":
			hosts = appendLauncherConsoleHostList(hosts, seen, ipv4s)
		case ip != nil && ip.To4() != nil:
			if hasIPv4Any {
				continue
			}
			hosts = appendUniqueHost(hosts, seen, ip.String())
		case ip != nil:
			if hasIPv6Any {
				continue
			}
			if isConsoleDisplayGlobalIPv6(ip) {
				hosts = appendUniqueHost(hosts, seen, ip.String())
			}
		default:
			hosts = appendUniqueHost(hosts, seen, token)
		}
	}

	return hosts
}

func launcherConsoleHosts(hostInput string, public bool) []string {
	return launcherConsoleHostsWithLocalAddrs(
		hostInput,
		public,
		utils.GetLocalIPv4s(),
		utils.GetGlobalIPv6s(),
	)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

type launcherAllowlistBypassLogDecision struct {
	level   logger.LogLevel
	message string
	emit    bool
}

func launcherBindMayExposeBeyondLoopback(hostInput string, public bool) bool {
	normalizedHostInput := strings.TrimSpace(hostInput)
	if normalizedHostInput == "" {
		return public
	}

	for token := range strings.SplitSeq(normalizedHostInput, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if token == "*" || netbind.IsUnspecifiedHost(token) {
			return true
		}
		if strings.EqualFold(token, "localhost") || netbind.IsLoopbackHost(token) {
			continue
		}
		return true
	}

	return false
}

func launcherAllowlistBypassLogPolicy(
	hostInput string,
	public bool,
	cfg launcherconfig.Config,
) launcherAllowlistBypassLogDecision {
	if !launcherBindMayExposeBeyondLoopback(hostInput, public) || len(cfg.AllowedCIDRs) == 0 {
		return launcherAllowlistBypassLogDecision{}
	}

	switch cfg.AllowLocalhostBypassSource {
	case launcherconfig.BoolFieldPresent:
		if cfg.AllowLocalhostBypass {
			return launcherAllowlistBypassLogDecision{
				level:   logger.INFO,
				emit:    true,
				message: "Launcher public access uses allowed_cidrs with allow_localhost_bypass=true; same-host proxies or tunnels can bypass CIDR restrictions",
			}
		}
	case launcherconfig.BoolFieldNull:
		return launcherAllowlistBypassLogDecision{
			level:   logger.WARN,
			emit:    true,
			message: "Launcher public access uses allowed_cidrs with allow_localhost_bypass=null; default localhost bypass remains enabled, so same-host proxies or tunnels can bypass CIDR restrictions",
		}
	}

	return launcherAllowlistBypassLogDecision{}
}

func main() {
	androiddns.ConfigureDefaultResolverFromEnvironment()

	port := flag.String("port", "18800", "Port to listen on")
	host := flag.String("host", "", "Host to listen on (overrides -public when set)")
	public := flag.Bool("public", false, "Listen on all interfaces (dual-stack) instead of localhost only")
	noBrowser = flag.Bool("no-browser", false, "Do not auto-open browser on startup")
	lang := flag.String("lang", "", "Language: en (English) or zh (Chinese). Default: auto-detect from system locale")
	console := flag.Bool("console", false, "Console mode, no GUI")

	var debug bool
	flag.BoolVar(&debug, "d", false, "Enable debug logging")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s Launcher - Web console and gateway manager\n\n", appName)
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [config.json]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  config.json    Path to the configuration file (default: ~/.picoclaw/config.json)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "      Use default config path in GUI mode\n")
		fmt.Fprintf(os.Stderr, "  %s ./config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "      Specify a config file\n")
		fmt.Fprintf(
			os.Stderr,
			"  %s -public ./config.json\n",
			os.Args[0],
		)
		fmt.Fprintf(os.Stderr, "      Allow access from other devices on the local network\n")
		fmt.Fprintf(os.Stderr, "  %s -host :: ./config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "      Bind launcher host explicitly with exact host semantics\n")
		fmt.Fprintf(os.Stderr, "  %s -console -d ./config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "      Run in the terminal with debug logs enabled\n")
	}
	flag.Parse()

	// Initialize logger
	picoHome := utils.GetPicoclawHome()

	f := filepath.Join(config.ResolveLogDir(picoHome), panicFile)
	panicFunc, err := logger.InitPanic(f)
	if err != nil {
		panic(fmt.Sprintf("error initializing panic log: %v", err))
	}
	defer panicFunc()

	enableConsole := *console
	fileLoggingEnabled := shouldEnableLauncherFileLogging(enableConsole, debug)
	if fileLoggingEnabled {
		// GUI mode writes launcher logs to file. Debug mode keeps file logging enabled in console mode too.
		if !debug {
			logger.DisableConsole()
		}

		f := filepath.Join(config.ResolveLogDir(picoHome), logFile)
		if err = logger.EnableFileLogging(f); err != nil {
			panic(fmt.Sprintf("error enabling file logging: %v", err))
		}
		defer logger.DisableFileLogging()
	}
	if debug {
		logger.SetLevel(logger.DEBUG)
	}

	// Set language from command line or auto-detect
	if *lang != "" {
		SetLanguage(*lang)
	}

	// Resolve config path
	configPath := utils.GetDefaultConfigPath()
	if flag.NArg() > 0 {
		configPath = flag.Arg(0)
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		logger.Fatalf("Failed to resolve config path: %v", err)
	}
	err = utils.EnsureOnboarded(absPath)
	if err != nil {
		logger.Errorf("Warning: Failed to initialize %s config automatically: %v", appName, err)
	}
	if !debug {
		logger.SetLevelFromString(config.ResolveGatewayLogLevel(absPath))
	}

	logger.InfoC("web", fmt.Sprintf("%s launcher starting (version %s)...", appName, appVersion))
	// The source this binary was built from. This is also what keeps
	// coresource.Stamped referenced by live code: the linker drops an unused
	// variable and takes the -X value with it, which would leave the staged
	// dashboard unverifiable while the build reported success.
	logger.InfoC("web", fmt.Sprintf("%s core source: %s", appName, coresource.Describe()))
	logger.InfoC("web", fmt.Sprintf("%s Home: %s", appName, picoHome))
	if debug {
		logger.InfoC("web", "Debug mode enabled")
		logger.DebugC(
			"web",
			fmt.Sprintf(
				"Launcher flags: console=%t host=%q public=%t no_browser=%t config=%s",
				enableConsole,
				*host,
				*public,
				*noBrowser,
				absPath,
			),
		)
	}

	explicitPort, explicitHost, explicitPublic := launcherExplicitFlags(flag.CommandLine)

	launcherPath := launcherconfig.PathForAppConfig(absPath)
	launcherCfg, err := launcherconfig.Load(launcherPath, launcherconfig.Default())
	if err != nil {
		logger.ErrorC("web", fmt.Sprintf("Warning: Failed to load %s: %v", launcherPath, err))
		launcherCfg = launcherconfig.Default()
	}

	effectivePort := *port
	if !explicitPort {
		effectivePort = strconv.Itoa(launcherCfg.Port)
	}
	effectivePublic := resolveLauncherPublicMode(*public, explicitPublic, launcherCfg.Public)
	envHost := strings.TrimSpace(os.Getenv(launcherconfig.EnvLauncherHost))

	hostInput, hostOverrideActive, err := resolveLauncherHostInput(*host, explicitHost, envHost)
	if err != nil {
		logger.Fatalf("Invalid host %q: %v", firstNonEmpty(strings.TrimSpace(*host), envHost), err)
	}
	if hostOverrideActive {
		effectivePublic = false
	}

	if !explicitHost && hostOverrideActive {
		logger.InfoC("web", "Using launcher host from the environment")
	}

	if hostOverrideActive && explicitPublic {
		logger.InfoC("web", "Ignoring -public because launcher host was explicitly set")
	}

	if decision := launcherAllowlistBypassLogPolicy(hostInput, effectivePublic, launcherCfg); decision.emit {
		switch decision.level {
		case logger.WARN:
			logger.WarnC("web", decision.message)
		default:
			logger.InfoC("web", decision.message)
		}
	}

	portNum, err := strconv.Atoi(effectivePort)
	if err != nil || portNum < 1 || portNum > 65535 {
		if err == nil {
			err = errors.New("must be in range 1-65535")
		}
		logger.Fatalf("Invalid port %q: %v", effectivePort, err)
	}

	dashboardSessions := middleware.NewLauncherDashboardSessions(0)

	// The credential verifier moves to private storage where the host demands
	// one. Migration runs before the store is opened so an existing password
	// survives the move.
	//
	// When PICOCLAW_DASHBOARD_AUTH_DIR is set it is a security boundary, not a
	// preference: on Android the shared location is writable by any app with
	// storage access, and an attacker who replaces the stored verifier there
	// can log in with a password of their choosing. So a failure to establish
	// the private store fails startup rather than falling back — falling back
	// would re-arm precisely the vector the override exists to remove. The
	// user's password is never reset and the legacy database is left intact for
	// recovery.
	dashboardAuthDir := config.ResolveDashboardAuthDir(picoHome)
	privateAuthRequired := config.DashboardAuthDirOverridden()
	migration, migrationErr := dashboardauth.MigrateLegacyDatabase(
		context.Background(), picoHome, dashboardAuthDir,
	)
	switch {
	case migrationErr != nil && privateAuthRequired:
		logger.Fatalf(
			"Dashboard authentication requires private storage and it could not be "+
				"established at %s: %v. The existing credential database was left "+
				"untouched; shared storage will not be used as the active verifier.",
			dashboardAuthDir, migrationErr)
	case migrationErr != nil:
		// No private storage was demanded, so this is the historical desktop
		// and server behaviour: keep using the home directory.
		logger.ErrorC("web", fmt.Sprintf(
			"Dashboard credential migration skipped, continuing with the existing store: %v",
			migrationErr))
		dashboardAuthDir = picoHome
	case !migration.PrivateReady && privateAuthRequired:
		logger.Fatalf(
			"Dashboard authentication requires private storage at %s and it is not "+
				"ready (%s). Shared storage will not be used as the active verifier.",
			dashboardAuthDir, migration.Reason)
	case migration.Migrated:
		logger.InfoC("web", fmt.Sprintf(
			"Migrated the Dashboard credential store to private storage (legacy removed: %t)",
			migration.LegacyRemoved))
	case migration.LegacyRemoved:
		logger.InfoC("web",
			"Removed the superseded Dashboard credential database from shared storage")
	}

	// Open the bcrypt password store (creates the DB file on first run).
	authStore, authStoreErr := dashboardauth.New(dashboardAuthDir)
	var passwordStore api.PasswordStore
	if authStoreErr == nil {
		passwordStore = authStore
		defer authStore.Close()
	} else if errors.Is(authStoreErr, dashboardauth.ErrUnsupportedPlatform) {
		logger.InfoC(
			"web",
			fmt.Sprintf(
				"Dashboard SQLite password store unavailable on this platform; using launcher-config password storage: %v",
				authStoreErr,
			),
		)
		passwordStore = launcherconfig.NewPasswordStore(launcherPath, launcherCfg)
		authStoreErr = nil
	} else {
		logger.ErrorC("web", fmt.Sprintf("Warning: could not open auth store: %v", authStoreErr))
	}

	migrationResult, migrationErr := launcherconfig.MigrateLegacyLauncherToken(
		context.Background(),
		passwordStore,
		launcherPath,
		launcherCfg,
	)
	if migrationErr != nil {
		logger.Fatalf("Failed to migrate legacy launcher token to password login: %v", migrationErr)
	}
	if migrationResult.Migrated {
		logger.InfoC("web", "Migrated legacy launcher token to dashboard password login")
	}
	if migrationResult.CleanupErr != nil {
		logger.WarnC(
			"web",
			fmt.Sprintf(
				"Legacy launcher token password migration succeeded, but failed to remove launcher_token from %s: %v",
				launcherPath,
				migrationResult.CleanupErr,
			),
		)
	}

	// PC-DEF-039. An unclaimed dashboard is never exposed beyond loopback.
	//
	// Public Mode says where the user wants the dashboard reachable from. It
	// does not say who owns it, and until a password exists nobody does: the
	// first client to reach POST /api/auth/setup would become the owner. The
	// handler refuses that from off-device, and this refuses to offer them the
	// port in the first place -- two independent controls, because either one
	// regressing alone must not reopen the takeover.
	//
	// The desired preference is untouched and is reported as desired; only the
	// effective bind is narrowed while the dashboard is unclaimed.
	desiredPublic := effectivePublic
	dashboardInitialized := false
	if passwordStore != nil {
		if ok, initErr := passwordStore.IsInitialized(context.Background()); initErr != nil {
			logger.ErrorC("web", fmt.Sprintf(
				"Could not determine dashboard initialization state, binding to loopback only: %v", initErr))
		} else {
			dashboardInitialized = ok
		}
	}
	effectivePublic, narrowed := effectiveLauncherExposure(desiredPublic, dashboardInitialized)
	if narrowed {
		logger.WarnC("web",
			"Public Mode is requested but the Dashboard has no password yet; "+
				"binding to loopback only until it is set up on this device")
	}

	openResult, err := openLauncherListeners(hostInput, effectivePublic, effectivePort)
	if err != nil {
		logger.Fatalf("Failed to open launcher listener(s): %v", err)
	}
	listeners := openResult.Listeners

	var localAutoLogin *middleware.LauncherDashboardLocalAutoLogin
	needsInitialSetup := false
	if passwordStore != nil {
		initialized, initErr := passwordStore.IsInitialized(context.Background())
		if initErr != nil {
			logger.ErrorC("web", fmt.Sprintf("Warning: could not check dashboard password state: %v", initErr))
		} else if !initialized {
			needsInitialSetup = true
		} else if shouldEnableLocalAutoLogin(*noBrowser, openResult.ProbeHost) {
			localAutoLogin, err = middleware.NewLauncherDashboardLocalAutoLogin(5 * time.Minute)
			if err != nil {
				logger.Fatalf("Failed to create local auto-login grant: %v", err)
			}
		}
	}

	// Initialize Server components
	mux := http.NewServeMux()

	api.RegisterLauncherAuthRoutes(mux, api.LauncherAuthRouteOpts{
		Sessions:      dashboardSessions,
		PasswordStore: passwordStore,
		StoreError:    authStoreErr,
		// PC-DEF-040. Resolved when called, not now: the routes are registered
		// before the HTTP runtime exists, and the runtime is what owns the
		// listener this re-binds.
		OnDashboardClaimed: func() {
			runtime := httpRuntime
			if runtime == nil {
				return
			}
			// Returns immediately: the apply happens on its own goroutine
			// because it replaces the listener carrying this very request.
			runtime.ReconcileAfterDashboardClaimed()
		},
	})

	// API Routes (e.g. /api/status)
	apiHandler = api.NewHandler(absPath)
	apiHandler.SetDebug(debug)
	if _, err = apiHandler.EnsurePocketClawChannel(); err != nil {
		logger.ErrorC("web", fmt.Sprintf("Warning: failed to ensure pico channel on startup: %v", err))
	}
	apiHandler.SetServerOptions(portNum, effectivePublic, explicitPublic, launcherCfg.AllowedCIDRs)
	apiHandler.SetServerAccessOptions(
		launcherCfg.AllowLocalhostBypass,
		launcherCfg.TrustedProxyCIDRs,
	)
	apiHandler.SetServerBindHost(hostInput, hostOverrideActive)
	apiHandler.RegisterRoutes(mux)

	// Frontend Embedded Assets
	registerEmbedRoutes(mux)

	accessControlledMux, err := middleware.IPAllowlist(middleware.IPAllowlistConfig{
		AllowedCIDRs:         launcherCfg.AllowedCIDRs,
		AllowLocalhostBypass: launcherCfg.AllowLocalhostBypass,
		TrustedProxyCIDRs:    launcherCfg.TrustedProxyCIDRs,
	}, mux)
	if err != nil {
		logger.Fatalf("Invalid allowed CIDR configuration: %v", err)
	}

	dashAuth := middleware.LauncherDashboardAuth(middleware.LauncherDashboardAuthConfig{
		Sessions:       dashboardSessions,
		LocalAutoLogin: localAutoLogin,
	}, accessControlledMux)
	appMux := http.NewServeMux()
	apiHandler.RegisterAndroidBridgeRoutes(appMux, os.Getenv(api.AndroidBridgeTokenEnv))
	// PC-DEF-030. One internal notification, authorized by a credential that
	// belongs to the current gateway generation and grants nothing else.
	apiHandler.RegisterGatewayIdleRoute(appMux)
	appMux.Handle("/", dashAuth)

	// Apply middleware stack
	handler := middleware.Recoverer(
		middleware.Logger(
			middleware.ReferrerPolicyNoReferrer(
				middleware.JSONContentType(appMux),
			),
		),
	)
	httpRuntime = newLauncherHTTPRuntime(
		handler, hostInput, effectivePublic, desiredPublic, openResult)
	apiHandler.SetLauncherNetworkModeController(httpRuntime)

	// Print startup banner (console mode only). Android captures stdout for a
	// plain-text Logs screen, so its NO_COLOR environment gets a text banner
	// rather than RGB escapes and terminal-only block drawing characters.
	if enableConsole || debug {
		consoleHosts := launcherConsoleHosts(hostInput, effectivePublic)

		if os.Getenv("NO_COLOR") != "" {
			fmt.Println(utils.PlainBanner)
		} else {
			fmt.Print(utils.Banner)
		}
		fmt.Println()
		if needsInitialSetup {
			if *noBrowser {
				fmt.Println("  First-time setup: open /launcher-setup to create the dashboard password.")
			} else {
				fmt.Println("  Launcher will open /launcher-setup automatically.")
			}
			fmt.Println()
		}
		fmt.Println("  Dashboard address:")
		fmt.Println()
		for _, host := range consoleHosts {
			fmt.Printf("    >> http://%s <<\n", net.JoinHostPort(host, effectivePort))
		}
		fmt.Println()
	}

	// Log startup info to file
	for _, ln := range listeners {
		logger.InfoC("web", fmt.Sprintf("Server will listen on http://%s", ln.Addr().String()))
	}
	if hasWildcardBindHosts(openResult.BindHosts) {
		if ip := advertiseIPForWildcardBindHosts(openResult.BindHosts); ip != "" {
			logger.InfoC("web", fmt.Sprintf("Public access enabled at http://%s", net.JoinHostPort(ip, effectivePort)))
		}
	}

	// Share the local URL with the launcher runtime.
	serverAddr = fmt.Sprintf("http://%s", net.JoinHostPort(openResult.ProbeHost, effectivePort))
	browserLaunchURL = serverAddr + launcherBrowserLaunchSuffix(needsInitialSetup, localAutoLogin)

	// Auto-open browser will be handled by the launcher runtime.

	// Auto-start gateway after backend starts listening, unless the host asked
	// to leave Gateway startup under the user's control.
	if gatewayAutoStartEnabled(os.Getenv("POCKETCLAW_GATEWAY_AUTOSTART")) {
		go func() {
			time.Sleep(1 * time.Second)
			apiHandler.TryAutoStartGateway()
		}()
	} else {
		logger.InfoC("gateway", "Gateway auto-start skipped: disabled by launch preference")
	}

	// Start the authenticated Dashboard listener(s). The runtime can replace
	// only these listeners when Android applies Public Mode; Core stays alive.
	httpRuntime.Start()

	defer shutdownApp()

	// Start system tray or run in console mode
	if enableConsole {
		if !*noBrowser {
			// Auto-open browser after systray is ready (if not disabled)
			// Check no-browser flag via environment or pass as parameter if needed
			if err := openBrowser(); err != nil {
				logger.Errorf("Warning: Failed to auto-open browser: %v", err)
			}
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Main event loop - wait for signals or config changes
		for {
			select {
			case <-sigChan:
				logger.Info("Shutting down...")

				return
			}
		}
	} else {
		// GUI mode: start system tray
		runTray()
	}
}
