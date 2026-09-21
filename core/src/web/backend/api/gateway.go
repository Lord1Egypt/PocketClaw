package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/health"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/netbind"
	ppid "github.com/sipeed/picoclaw/pkg/pid"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/web/backend/utils"
)

// gateway holds the state for the managed gateway process.
var gateway = struct {
	mu                  sync.Mutex
	cmd                 *exec.Cmd
	owned               bool // true if we started the process, false if we attached to an existing one
	bootDefaultModel    string
	bootConfigSignature string
	runtimeStatus       string
	startupDeadline     time.Time
	logs                *LogBuffer
	pidData             *ppid.PidFileData // pid file data read from .pocketclaw.pid
	pocketClawToken     string            // cached raw PocketClaw token for gateway proxy injection
	lastStartupError    string            // sanitized reason the last start attempt failed
}{
	runtimeStatus: "stopped",
	logs:          NewLogBuffer(200),
}

// refreshPocketClawTokensLocked reads the PocketClaw token from config and caches it.
// Caller must hold gateway.mu (or be sole writer).
func refreshPocketClawTokensLocked(configPath string) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return
	}
	var pocketClawCfg config.PocketClawSettings
	if bc := cfg.Channels.GetByType(config.ChannelPocketClaw); bc != nil {
		decoded, err := bc.GetDecoded()
		if err == nil && decoded != nil {
			if p, ok := decoded.(*config.PocketClawSettings); ok {
				pocketClawCfg = *p
			}
		}
	}
	gateway.pocketClawToken = pocketClawCfg.Token.String()
}

// ensurePocketClawTokenCachedLocked lazily fills the in-memory PocketClaw token cache when
// the launcher has already discovered a running gateway via pidData, but has
// not yet refreshed the token into memory.
func ensurePocketClawTokenCachedLocked(configPath string) {
	if gateway.pocketClawToken != "" {
		return
	}
	refreshPocketClawTokensLocked(configPath)
}

func (h *Handler) gatewayCommandArgs() []string {
	// The gateway's stdout/stderr are captured into user-facing Web Console
	// logs, never attached to an interactive terminal.
	args := []string{"gateway", "-E", "--no-color"}
	if h.debug {
		args = append(args, "-d")
	}
	return args
}

const (
	protocolKey = "Sec-Websocket-Protocol"
	tokenPrefix = "token."
)

// pocketClawGatewayProtocol returns the gateway-facing PocketClaw subprotocol that the
// launcher should inject when proxying browser traffic upstream.
func pocketClawGatewayProtocol() string {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()
	if gateway.pocketClawToken == "" {
		return ""
	}
	return tokenPrefix + gateway.pocketClawToken
}

var (
	gatewayStartupWindow          = 15 * time.Second
	gatewayRestartGracePeriod     = 5 * time.Second
	gatewayRestartForceKillWindow = 3 * time.Second
	gatewayRestartPollInterval    = 100 * time.Millisecond
	gatewayExecCommand            = exec.Command
)

var gatewayHealthGet = func(url string, timeout time.Duration) (*http.Response, error) {
	client := http.Client{Timeout: timeout}
	return client.Get(url)
}

var gatewayProcessMatcher = isLikelyGatewayProcess

// getGatewayHealth checks the gateway health endpoint and returns the status response.
// Returns (*health.StatusResponse, statusCode, error). If error is not nil, the other values are not valid.
func (h *Handler) getGatewayHealth(cfg *config.Config, timeout time.Duration) (*health.StatusResponse, int, error) {
	// Prefer port/host from pidData when available.
	var port int
	var host string
	gateway.mu.Lock()
	if d := gateway.pidData; d != nil && d.Port > 0 {
		port = d.Port
		host = gatewayProbeHost(d.Host)
	}
	gateway.mu.Unlock()
	if port == 0 {
		port = 18790
		if cfg != nil && cfg.Gateway.Port != 0 {
			port = cfg.Gateway.Port
		}
	}
	if host == "" {
		host = gatewayProbeHost(h.effectiveGatewayBindHost(cfg))
	}

	url := "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/health"

	return getGatewayHealthByURL(url, timeout)
}

func getGatewayHealthByURL(url string, timeout time.Duration) (*health.StatusResponse, int, error) {
	resp, err := gatewayHealthGet(url, timeout)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	var healthResponse health.StatusResponse
	if decErr := json.NewDecoder(resp.Body).Decode(&healthResponse); decErr != nil {
		return nil, resp.StatusCode, decErr
	}

	return &healthResponse, resp.StatusCode, nil
}

// isLikelyGatewayProcess returns whether PID appears to be a picoclaw gateway
// process plus whether inspection was conclusive on this platform/environment.
func isLikelyGatewayProcess(pid int) (bool, bool) {
	if pid <= 0 {
		return false, true
	}

	if runtime.GOOS == "windows" {
		psCmd := fmt.Sprintf(
			`$p=Get-CimInstance Win32_Process -Filter "ProcessId = %d"; if ($null -eq $p) { "" } else { $p.CommandLine }`,
			pid,
		)
		out, err := launcherExecCommand("powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
		if err == nil {
			cmdline := strings.TrimSpace(string(out))
			if cmdline != "" {
				return looksLikeGatewayCommandLine(cmdline), true
			}
		}

		// Fallback: determine only whether the process still exists.
		out, err = launcherExecCommand("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/FO", "CSV", "/NH").Output()
		if err != nil {
			return false, false
		}
		line := strings.ToLower(strings.TrimSpace(string(out)))
		if line == "" {
			return false, true
		}
		// A CSV row means the process exists, but may have a custom executable
		// name we cannot classify here.
		if strings.HasPrefix(line, "\"") {
			if strings.Contains(line, "\"picoclaw.exe\"") {
				return true, true
			}
			return false, true
		}
		if strings.Contains(line, "no tasks are running") {
			return false, true
		}
		return false, true
	}

	out, err := launcherExecCommand("ps", "-o", "command=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return false, false
	}
	return classifyGatewayCommandLine(string(out))
}

// classifyGatewayCommandLine turns one line of `ps` output into the
// (isGateway, inspected) verdict.
//
// A positive match proves the process is a gateway. A negative one only
// disproves it when `ps` actually returned a command line. Android's toybox
// reports the bare executable name for a launcher-spawned child, so the
// `gateway` subcommand is absent from output that is otherwise valid. Reading
// that as proof of a foreign process is what deleted a live gateway's pid
// file, so a bare name is reported as un-inspected and the health probe
// decides instead.
func classifyGatewayCommandLine(psOutput string) (isGateway bool, inspected bool) {
	cmdline := strings.ToLower(strings.TrimSpace(psOutput))
	if cmdline == "" {
		return false, true
	}
	if looksLikeGatewayCommandLine(cmdline) {
		return true, true
	}
	if len(strings.Fields(cmdline)) < 2 {
		return false, false
	}
	return false, true
}

// looksLikeGatewayCommandLine checks whether a process command line likely
// represents "picoclaw gateway ..." regardless of executable filename.
func looksLikeGatewayCommandLine(cmdline string) bool {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(cmdline)))
	if len(fields) == 0 {
		return false
	}
	for _, f := range fields {
		token := strings.Trim(f, `"'`)
		if token == "gateway" || strings.HasSuffix(token, "/gateway") || strings.HasSuffix(token, `\gateway`) {
			return true
		}
	}
	return false
}

func (h *Handler) getGatewayHealthForPidData(
	pidData *ppid.PidFileData,
	cfg *config.Config,
	timeout time.Duration,
) (*health.StatusResponse, int, error) {
	if pidData == nil {
		return nil, 0, errors.New("nil pid data")
	}

	port := pidData.Port
	if port == 0 {
		port = 18790
		if cfg != nil && cfg.Gateway.Port != 0 {
			port = cfg.Gateway.Port
		}
	}

	host := gatewayProbeHost(strings.TrimSpace(pidData.Host))
	if host == "" {
		host = gatewayProbeHost(h.effectiveGatewayBindHost(cfg))
	}
	if host == "" {
		host = netbind.ResolveAdaptiveLoopbackHost()
	}

	url := "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/health"
	return getGatewayHealthByURL(url, timeout)
}

// launcherOwnsGatewayPID reports whether this launcher spawned the process the
// pid file names and that process is still alive.
//
// This outranks command-line inspection: exec.Cmd ownership is first-hand
// evidence, while `ps` output is a platform-dependent description. It is also
// safe against pid reuse — the child stays a zombie, holding its pid, until
// cmd.Wait() returns, and the monitor goroutine clears gateway.cmd at that
// moment.
func launcherOwnsGatewayPID(pid int) bool {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	if !gateway.owned || gateway.cmd == nil || gateway.cmd.Process == nil {
		return false
	}
	if gateway.cmd.Process.Pid != pid {
		return false
	}
	return isCmdProcessAliveLocked(gateway.cmd)
}

// gatewayHealthProbeRefused reports whether a health probe failed because
// nothing is listening on the gateway port.
//
// The gateway opens its listeners before it commits the pid file, and closes
// them again when the singleton check rejects the start, so a process that
// legitimately owns the pid file is always accepting connections on that port.
// A refused connection is therefore decisive proof that the pid file is stale.
// A timeout is not: a wedged or merely busy gateway can leave a probe hanging,
// and deleting its pid file would be the bug this guard exists to prevent.
func gatewayHealthProbeRefused(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return false
	}
	return errors.Is(err, syscall.ECONNREFUSED)
}

func (h *Handler) validateGatewayPidData(
	pidData *ppid.PidFileData,
	cfg *config.Config,
	stage string,
) (ok bool, decisive bool, reason string) {
	if pidData == nil || pidData.PID <= 0 {
		return false, true, "invalid pid data"
	}

	if launcherOwnsGatewayPID(pidData.PID) {
		logGatewayPidValidation(pidData.PID, "owned", "launcher_spawned", stage, "")
		return true, true, ""
	}

	if gatewayProcess, inspected := gatewayProcessMatcher(pidData.PID); inspected {
		if !gatewayProcess {
			logGatewayPidValidation(pidData.PID, "foreign", "command_line", stage, "command line is not a gateway")
			return false, true, "pid belongs to another process; ignoring stale pid file"
		}
		logGatewayPidValidation(pidData.PID, "owned", "command_line", stage, "")
		return true, true, ""
	}

	healthResp, statusCode, err := h.getGatewayHealthForPidData(pidData, cfg, 800*time.Millisecond)
	if err != nil {
		if gatewayHealthProbeRefused(err) {
			logGatewayPidValidation(pidData.PID, "foreign", "health_probe", stage, "no listener on the gateway port")
			return false, true, fmt.Sprintf("health probe refused: %v", err)
		}
		logGatewayPidValidation(pidData.PID, "unknown", "health_probe", stage, "health endpoint unreachable")
		return false, false, fmt.Sprintf("health probe failed: %v", err)
	}
	if statusCode != http.StatusOK {
		logGatewayPidValidation(pidData.PID, "unknown", "health_probe", stage, "health endpoint not ok")
		return false, false, fmt.Sprintf("health endpoint returned status %d", statusCode)
	}
	if healthResp.PID > 0 && healthResp.PID != pidData.PID {
		logGatewayPidValidation(pidData.PID, "foreign", "health_identity", stage, "health pid does not match the pid file")
		return false, true, fmt.Sprintf("health pid mismatch: pidFile=%d, health=%d", pidData.PID, healthResp.PID)
	}
	logGatewayPidValidation(pidData.PID, "owned", "health_identity", stage, "")
	return true, true, ""
}

// gatewayPidValidationState suppresses repeats so a status poll cannot emit the
// same validation line every few seconds. Only transitions are logged.
var gatewayPidValidationState = struct {
	mu     sync.Mutex
	pid    int
	result string
	signal string
}{}

// The caller has already been through ReadPidFileWithCheck, which removes the
// pid file for a process that is not running, so anything reaching validation
// is alive and names itself in the pid file.
func logGatewayPidValidation(pid int, result, signal, stage, reason string) {
	gatewayPidValidationState.mu.Lock()
	repeat := gatewayPidValidationState.pid == pid &&
		gatewayPidValidationState.result == result &&
		gatewayPidValidationState.signal == signal
	gatewayPidValidationState.pid = pid
	gatewayPidValidationState.result = result
	gatewayPidValidationState.signal = signal
	gatewayPidValidationState.mu.Unlock()
	if repeat {
		return
	}

	fields := map[string]any{
		"event":             "gateway.pid.validation",
		"pid":               pid,
		"process_alive":     true,
		"pidfile_match":     true,
		"ownership_signal":  signal,
		"validation_result": result,
		"stage":             stage,
	}
	if reason != "" {
		fields["reason"] = reason
	}
	logger.DebugCF("gateway", "Gateway PID validation", fields)
}

func (h *Handler) sanitizeGatewayPidData(
	pidData *ppid.PidFileData,
	cfg *config.Config,
	stage string,
) *ppid.PidFileData {
	if pidData == nil {
		return nil
	}

	ok, decisive, reason := h.validateGatewayPidData(pidData, cfg, stage)
	if ok {
		return pidData
	}

	logger.Warnf("ignore pid file for PID %d: %s", pidData.PID, reason)
	if decisive && ppid.RemovePidFileIfPID(globalConfigDir(), pidData.PID) {
		logger.Warnf("removed stale pid file for PID %d", pidData.PID)
	}
	return nil
}

// registerGatewayRoutes binds gateway lifecycle endpoints to the ServeMux.
func (h *Handler) registerGatewayRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/gateway/status", h.handleGatewayStatus)
	mux.HandleFunc("GET /api/gateway/logs", h.handleGatewayLogs)
	mux.HandleFunc("POST /api/gateway/logs/clear", h.handleGatewayClearLogs)
	mux.HandleFunc("POST /api/gateway/start", h.handleGatewayStart)
	mux.HandleFunc("POST /api/gateway/stop", h.handleGatewayStop)
	mux.HandleFunc("POST /api/gateway/restart", h.handleGatewayRestart)
	mux.HandleFunc("POST /api/gateway/apply-config", h.handleGatewayApplyConfig)
}

// TryAutoStartGateway checks whether gateway start preconditions are met and
// starts it when possible. Intended to be called by the backend at startup.
func (h *Handler) TryAutoStartGateway() {
	// Check PID file first to detect an already-running gateway.
	pidData := h.sanitizeGatewayPidData(ppid.ReadPidFileWithCheck(globalConfigDir()), nil, "autostart")
	if pidData != nil {
		gateway.mu.Lock()
		ready, reason, err := h.gatewayInfrastructureReady()
		if err != nil {
			logger.ErrorC("gateway", fmt.Sprintf("Skip auto-starting gateway: %v", err))
			gateway.mu.Unlock()
			return
		}
		logger.Infof("ready: %v, reason: %s", ready, reason)
		if !ready {
			logger.InfoC("gateway", fmt.Sprintf("Skip auto-starting gateway: %s", reason))
			gateway.mu.Unlock()
			return
		}
		pid := pidData.PID
		_, err = h.startGatewayLocked("starting", pid)
		if err != nil {
			logger.ErrorC("gateway", fmt.Sprintf("Failed to attach to running gateway (PID: %d): %v", pid, err))
		} else {
			gateway.pidData = pidData
			refreshPocketClawTokensLocked(h.configPath)
			logger.InfoC("gateway", fmt.Sprintf("Attached to running gateway via PID file (PID: %d)", pid))
		}
		gateway.mu.Unlock()
		return
	}

	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	if gateway.cmd != nil && gateway.cmd.Process != nil {
		gateway.cmd = nil
	}

	ready, reason, err := h.gatewayInfrastructureReady()
	if err != nil {
		logger.ErrorC("gateway", fmt.Sprintf("Skip auto-starting gateway: %v", err))
		return
	}
	if !ready {
		logger.InfoC("gateway", fmt.Sprintf("Skip auto-starting gateway: %s", reason))
		return
	}

	pid, err := h.startGatewayLocked("starting", 0)
	if err != nil {
		logger.ErrorC("gateway", fmt.Sprintf("Failed to auto-start gateway: %v", err))
		return
	}
	logger.InfoC("gateway", fmt.Sprintf("Gateway auto-started (PID: %d)", pid))
}

// gatewayInfrastructureReady reports whether the gateway PROCESS can start.
//
// Infrastructure only. It must never consult provider, model, credential or
// reachability state: the gateway is infrastructure, and a PocketClaw with no
// AI provider configured is a valid running system whose dashboard, settings,
// provider setup and health endpoints all have to be reachable -- that is how
// a user configures their first provider in the first place.
//
// This function and chatReady were one function until PC-DEF-038. Conflating
// them made "add a model" a hidden prerequisite for "make the gateway start":
// a fresh install reported "Skip auto-starting gateway: no default model
// configured" and manual start returned precondition_failed, so the one path
// a new user has to reach provider setup was gated on already having done it.
func (h *Handler) gatewayInfrastructureReady() (bool, string, error) {
	if _, err := config.LoadConfig(h.configPath); err != nil {
		return false, "", fmt.Errorf("failed to load config: %w", err)
	}
	return true, "", nil
}

// chatReady reports whether an AI-dependent request can actually be served.
//
// This is the other half of the old gatewayStartReady: everything here is a
// real prerequisite for talking to a model, and none of it is a prerequisite
// for running the gateway. Callers that serve chat ask this; callers that
// start or restart the process must not.
func (h *Handler) chatReady() (bool, string, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return false, "", fmt.Errorf("failed to load config: %w", err)
	}

	modelName := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
	if modelName == "" {
		return false, "no default model configured", nil
	}

	modelCfg := lookupModelConfig(cfg, modelName)
	if modelCfg == nil {
		return false, fmt.Sprintf("default model %q is invalid", modelName), nil
	}
	if !defaultModelAllowedForModelConfig(modelCfg) {
		return false, fmt.Sprintf("default model %q is not usable for chat", modelName), nil
	}

	if !hasModelConfiguration(modelCfg) {
		return false, fmt.Sprintf("default model %q has no credentials configured", modelName), nil
	}
	if requiresRuntimeProbe(modelCfg) && !probeLocalModelAvailability(modelCfg) {
		return false, fmt.Sprintf("default model %q is not reachable", modelName), nil
	}

	return true, "", nil
}

func lookupModelConfig(cfg *config.Config, modelName string) *config.ModelConfig {
	modelCfg, err := cfg.GetModelConfig(modelName)
	if err != nil {
		return nil
	}
	return modelCfg
}

func computeConfigSignature(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	var parts []string
	defaultModel := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
	if defaultModel != "" {
		parts = append(parts, "model:"+defaultModel)
	}
	modelStreamingSignatures := computeModelStreamingSignatures(cfg)
	if len(modelStreamingSignatures) > 0 {
		parts = append(parts, "model_streaming:"+strings.Join(modelStreamingSignatures, ","))
	}
	modelCredentialSignatures := computeModelCredentialSignatures(cfg)
	if len(modelCredentialSignatures) > 0 {
		parts = append(parts, "model_credentials:"+strings.Join(modelCredentialSignatures, ","))
	}
	toolSignatures := []string{}
	if cfg.Tools.ReadFile.Enabled {
		toolSignatures = append(toolSignatures, "read_file")
	}
	if cfg.Tools.WriteFile.Enabled {
		toolSignatures = append(toolSignatures, "write_file")
	}
	if cfg.Tools.ListDir.Enabled {
		toolSignatures = append(toolSignatures, "list_dir")
	}
	if cfg.Tools.EditFile.Enabled {
		toolSignatures = append(toolSignatures, "edit_file")
	}
	if cfg.Tools.AppendFile.Enabled {
		toolSignatures = append(toolSignatures, "append_file")
	}
	if cfg.Tools.Exec.Enabled {
		toolSignatures = append(toolSignatures, "exec")
	}
	if cfg.Tools.Cron.Enabled {
		toolSignatures = append(toolSignatures, "cron")
	}
	if cfg.Tools.Web.Enabled {
		toolSignatures = append(toolSignatures, "web")
		// Digested, not embedded. This subtree holds every web-search
		// credential -- brave, tavily, kagi, perplexity, baidu, glm, gemini --
		// plus a proxy URL that can carry userinfo. See signature_digest.go.
		webConfig, err := json.Marshal(canonicalizeSignatureValue(reflect.ValueOf(cfg.Tools.Web)))
		if err == nil {
			parts = append(parts, "webcfg:"+signatureDigestBytes(webConfig))
		}
	}
	if cfg.Tools.WebFetch.Enabled {
		toolSignatures = append(toolSignatures, "web_fetch")
	}
	if cfg.Tools.Message.Enabled {
		toolSignatures = append(toolSignatures, "message")
	}
	if cfg.Tools.SendFile.Enabled {
		toolSignatures = append(toolSignatures, "send_file")
	}
	if cfg.Tools.FindSkills.Enabled {
		toolSignatures = append(toolSignatures, "find_skills")
	}
	if cfg.Tools.InstallSkill.Enabled {
		toolSignatures = append(toolSignatures, "install_skill")
	}
	if cfg.Tools.Spawn.Enabled {
		toolSignatures = append(toolSignatures, "spawn")
	}
	if cfg.Tools.SpawnStatus.Enabled {
		toolSignatures = append(toolSignatures, "spawn_status")
	}
	if cfg.Tools.I2C.Enabled {
		toolSignatures = append(toolSignatures, "i2c")
	}
	if cfg.Tools.SPI.Enabled {
		toolSignatures = append(toolSignatures, "spi")
	}
	if cfg.Tools.MCP.Enabled {
		toolSignatures = append(toolSignatures, "mcp")
	}
	if cfg.Tools.MCP.Discovery.Enabled {
		toolSignatures = append(toolSignatures, "mcp_discovery")
	}
	if cfg.Tools.MCP.Discovery.UseRegex {
		toolSignatures = append(toolSignatures, "mcp_discovery_regex")
	}
	if cfg.Tools.MCP.Discovery.UseBM25 {
		toolSignatures = append(toolSignatures, "mcp_discovery_bm25")
	}
	if len(toolSignatures) > 0 {
		parts = append(parts, "tools:"+strings.Join(toolSignatures, ","))
	}
	channelSignatures := computeChannelSignatures(cfg.Channels)
	if len(channelSignatures) > 0 {
		parts = append(parts, "channels:"+strings.Join(channelSignatures, ","))
	}
	return strings.Join(parts, ";")
}

func computeModelStreamingSignatures(cfg *config.Config) []string {
	if cfg == nil {
		return nil
	}
	defaultProvider := strings.TrimSpace(cfg.Agents.Defaults.Provider)
	if defaultProvider == "" {
		defaultProvider = "openai"
	}
	names := []string{strings.TrimSpace(cfg.Agents.Defaults.GetModelName())}
	names = append(names, cfg.Agents.Defaults.ModelFallbacks...)
	if cfg.Agents.Defaults.Routing != nil {
		names = append(names, cfg.Agents.Defaults.Routing.LightModel)
	}
	for _, agent := range cfg.Agents.List {
		if agent.Model == nil {
			continue
		}
		names = append(names, agent.Model.Primary)
		names = append(names, agent.Model.Fallbacks...)
	}

	seenNames := make(map[string]bool)
	seenEntries := make(map[string]bool)
	signatures := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seenNames[name] {
			continue
		}
		seenNames[name] = true
		for _, match := range modelConfigsMatchingSignatureRef(cfg.ModelList, name, defaultProvider) {
			mc := match.model
			entry := strings.Join([]string{
				name,
				strconv.Itoa(match.index),
				strings.TrimSpace(mc.Provider),
				strings.TrimSpace(mc.Model),
				strconv.FormatBool(mc.Streaming.Enabled),
			}, ":")
			if seenEntries[entry] {
				continue
			}
			seenEntries[entry] = true
			signatures = append(signatures, entry)
		}
	}
	sort.Strings(signatures)
	return signatures
}

type signatureModelConfigMatch struct {
	index int
	model *config.ModelConfig
}

func modelConfigsMatchingSignatureRef(
	modelList []*config.ModelConfig,
	raw string,
	defaultProvider string,
) []signatureModelConfigMatch {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	matches := make([]signatureModelConfigMatch, 0, 1)
	for i, mc := range modelList {
		if mc == nil || strings.TrimSpace(mc.ModelName) != raw {
			continue
		}
		matches = append(matches, signatureModelConfigMatch{index: i, model: mc})
	}
	if len(matches) > 0 {
		return matches
	}
	for i, mc := range modelList {
		if mc == nil || strings.TrimSpace(mc.Model) != raw {
			continue
		}
		return []signatureModelConfigMatch{{index: i, model: mc}}
	}
	for i, mc := range modelList {
		if modelConfigMatchesBareRef(mc, raw, defaultProvider) {
			return []signatureModelConfigMatch{{index: i, model: mc}}
		}
	}

	rawRef := providers.ParseModelRef(raw, "")
	rawHasProvider := rawRef != nil && hasUnambiguousProviderPrefix(raw) &&
		strings.TrimSpace(rawRef.Provider) != "" && strings.TrimSpace(rawRef.Model) != ""
	if rawHasProvider {
		for i, mc := range modelList {
			if modelConfigMatchesProviderRef(mc, raw) {
				return []signatureModelConfigMatch{{index: i, model: mc}}
			}
		}
	}
	return nil
}

func hasUnambiguousProviderPrefix(raw string) bool {
	provider, _, found := strings.Cut(strings.TrimSpace(raw), "/")
	if !found {
		return false
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return false
	}
	normalizedProvider := providers.NormalizeProvider(provider)
	if !providers.IsSupportedModelProvider(normalizedProvider) {
		return false
	}
	return true
}

func modelConfigMatchesProviderRef(mc *config.ModelConfig, raw string) bool {
	if mc == nil {
		return false
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	rawRef := providers.ParseModelRef(raw, "")
	if rawRef == nil || strings.TrimSpace(rawRef.Provider) == "" || strings.TrimSpace(rawRef.Model) == "" {
		return false
	}
	protocol, modelID := providers.ExtractProtocol(mc)
	return providers.ModelKey(protocol, modelID) == providers.ModelKey(rawRef.Provider, rawRef.Model)
}

func modelConfigMatchesBareRef(mc *config.ModelConfig, raw string, defaultProvider string) bool {
	if mc == nil {
		return false
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	protocol, modelID := providers.ExtractProtocol(mc)
	if strings.TrimSpace(modelID) != raw {
		return false
	}
	return providers.NormalizeProvider(protocol) == providers.NormalizeProvider(defaultProvider)
}

func computeChannelSignatures(channels config.ChannelsConfig) []string {
	if len(channels) == 0 {
		return nil
	}

	keys := make([]string, 0, len(channels))
	for name := range channels {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	signatures := make([]string, 0, len(keys))
	for _, name := range keys {
		channel := channels[name]
		if channel == nil {
			signatures = append(signatures, name+":<nil>")
			continue
		}

		payload := struct {
			Enabled            bool                       `json:"enabled"`
			Type               string                     `json:"type"`
			AllowFrom          config.FlexibleStringSlice `json:"allow_from,omitempty"`
			ReasoningChannelID string                     `json:"reasoning_channel_id,omitempty"`
			GroupTrigger       config.GroupTriggerConfig  `json:"group_trigger,omitempty"`
			Typing             config.TypingConfig        `json:"typing,omitempty"`
			Placeholder        config.PlaceholderConfig   `json:"placeholder,omitempty"`
			Settings           json.RawMessage            `json:"settings,omitempty"`
		}{
			Enabled:            channel.Enabled,
			Type:               channel.Type,
			AllowFrom:          channel.AllowFrom,
			ReasoningChannelID: channel.ReasoningChannelID,
			GroupTrigger:       channel.GroupTrigger,
			Typing:             channel.Typing,
			Placeholder:        channel.Placeholder,
			Settings:           normalizeChannelSettings(channel),
		}

		encoded, err := json.Marshal(payload)
		if err != nil {
			signatures = append(signatures, name+":<invalid>")
			continue
		}
		// Digested, not embedded. Settings carry the channel's credential --
		// a Telegram bot token, Slack bot/app tokens, a Matrix access token and
		// crypto passphrase, and so on -- and the raw-JSON fallback below dumps
		// them verbatim when a settings subtree cannot be decoded.
		signatures = append(signatures, name+":"+signatureDigestBytes(encoded))
	}

	return signatures
}

func normalizeChannelSettings(channel *config.Channel) json.RawMessage {
	if channel == nil {
		return nil
	}

	decoded, err := channel.GetDecoded()
	if err == nil && decoded != nil {
		normalized, err := json.Marshal(canonicalizeSignatureValue(reflect.ValueOf(decoded)))
		if err == nil {
			return normalized
		}
	}

	return normalizeRawJSON(channel.Settings)
}

func normalizeRawJSON(raw config.RawNode) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return bytes.TrimSpace(raw)
	}

	normalized, err := json.Marshal(value)
	if err != nil {
		return bytes.TrimSpace(raw)
	}
	return normalized
}

func canonicalizeSignatureValue(value reflect.Value) any {
	if !value.IsValid() {
		return nil
	}

	if value.CanInterface() {
		switch typed := value.Interface().(type) {
		case config.SecureString:
			return typed.String()
		case *config.SecureString:
			if typed == nil {
				return ""
			}
			return typed.String()
		case config.SecureStrings:
			return typed.Values()
		case *config.SecureStrings:
			if typed == nil {
				return nil
			}
			return typed.Values()
		}
	}

	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		return canonicalizeSignatureValue(value.Elem())
	case reflect.Struct:
		result := make(map[string]any)
		valueType := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := valueType.Field(i)
			if field.PkgPath != "" {
				continue
			}
			tag := field.Tag.Get("json")
			name := field.Name
			if tag != "" {
				if comma := strings.Index(tag, ","); comma >= 0 {
					tag = tag[:comma]
				}
				if tag == "-" {
					continue
				}
				if tag != "" {
					name = tag
				}
			}
			result[name] = canonicalizeSignatureValue(value.Field(i))
		}
		return result
	case reflect.Slice, reflect.Array:
		length := value.Len()
		result := make([]any, 0, length)
		for i := 0; i < length; i++ {
			result = append(result, canonicalizeSignatureValue(value.Index(i)))
		}
		return result
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return value.Interface()
		}
		result := make(map[string]any, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			result[iter.Key().String()] = canonicalizeSignatureValue(iter.Value())
		}
		return result
	default:
		if value.CanInterface() {
			return value.Interface()
		}
		return nil
	}
}

func gatewayRestartRequiredBySignature(bootSignature, currentSignature, gatewayStatus string) bool {
	if gatewayStatus != "running" {
		return false
	}
	if bootSignature == "" || currentSignature == "" {
		return false
	}
	return bootSignature != currentSignature
}

func isCmdProcessAliveLocked(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}

	// Wait() sets ProcessState when the process exits; use it when available.
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return false
	}

	// Windows does not support Signal(0) probing. If we still own cmd and it
	// has not reported exit, treat it as alive.
	if runtime.GOOS == "windows" {
		return true
	}

	err := cmd.Process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	var errno syscall.Errno
	// EPERM means the process exists but cannot be signaled by this user.
	return errors.As(err, &errno) && errno == syscall.EPERM
}

func setGatewayRuntimeStatusLocked(status string) {
	gateway.runtimeStatus = status
	// The failure reason is only meaningful while the gateway is in the error
	// state; any other transition clears it so a stale line cannot be shown.
	if status != "error" {
		gateway.lastStartupError = ""
	}
	if status == "starting" || status == "restarting" {
		gateway.startupDeadline = time.Now().Add(gatewayStartupWindow)
		return
	}
	gateway.startupDeadline = time.Time{}
}

// attachToGatewayProcess attaches to an existing gateway process by PID
// and updates the gateway state accordingly.
// Assumes gateway.mu is held by the caller.
func attachToGatewayProcessLocked(pid int, cfg *config.Config) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process for PID %d: %w", pid, err)
	}

	gateway.cmd = &exec.Cmd{Process: process}
	gateway.owned = false // We didn't start this process
	setGatewayRuntimeStatusLocked("running")

	// Update bootDefaultModel and bootConfigSignature from config
	if cfg != nil {
		defaultModelName := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
		gateway.bootDefaultModel = defaultModelName
		gateway.bootConfigSignature = computeConfigSignature(cfg)
	}

	logger.InfoC("gateway", fmt.Sprintf("Attached to gateway process (PID: %d)", pid))
	return nil
}

func gatewayStatusWithoutHealthLocked() string {
	if gateway.runtimeStatus == "starting" || gateway.runtimeStatus == "restarting" {
		if gateway.startupDeadline.IsZero() || time.Now().Before(gateway.startupDeadline) {
			return gateway.runtimeStatus
		}
		if gateway.lastStartupError == "" {
			gateway.lastStartupError = "Gateway did not become healthy within the startup window."
		}
		return "error"
	}
	if gateway.runtimeStatus == "running" {
		// For attached processes there is no waiter goroutine; degrade stale
		// running state once the tracked process exits.
		if !isCmdProcessAliveLocked(gateway.cmd) {
			gateway.cmd = nil
			gateway.owned = false
			gateway.bootDefaultModel = ""
			gateway.bootConfigSignature = ""
			return "stopped"
		}
		return "running"
	}
	if gateway.runtimeStatus == "error" {
		return "error"
	}
	return "stopped"
}

func waitForGatewayProcessExit(cmd *exec.Cmd, timeout time.Duration) bool {
	if cmd == nil || cmd.Process == nil {
		return true
	}

	deadline := time.Now().Add(timeout)
	for {
		if !isCmdProcessAliveLocked(cmd) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(gatewayRestartPollInterval)
	}
}

// StopGateway stops the gateway process if it was started by this handler.
// This method is called during application shutdown to ensure the gateway subprocess
// is properly terminated. It only stops processes that were started by this handler,
// not processes that were attached to from existing instances.
func (h *Handler) StopGateway() {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	// Only stop if we own the process (started it ourselves)
	if !gateway.owned || gateway.cmd == nil || gateway.cmd.Process == nil {
		return
	}

	pid, err := stopGatewayLocked()
	if err != nil {
		logger.ErrorC("gateway", fmt.Sprintf("Failed to stop gateway (PID %d): %v", pid, err))
		return
	}

	logger.InfoC("gateway", fmt.Sprintf("Gateway stopped (PID: %d)", pid))
}

// stopGatewayLocked sends a stop signal to the gateway process.
// Assumes gateway.mu is held by the caller.
// Returns the PID of the stopped process and any error encountered.
func stopGatewayLocked() (int, error) {
	if gateway.cmd == nil || gateway.cmd.Process == nil {
		return 0, nil
	}

	pid := gateway.cmd.Process.Pid
	if !gateway.owned {
		if isGateway, inspected := gatewayProcessMatcher(pid); inspected && !isGateway {
			return pid, fmt.Errorf("refuse to stop non-gateway process (PID %d)", pid)
		}
	}

	// Send SIGTERM for graceful shutdown (SIGKILL on Windows)
	var sigErr error
	if runtime.GOOS == "windows" {
		sigErr = gateway.cmd.Process.Kill()
	} else {
		sigErr = gateway.cmd.Process.Signal(syscall.SIGTERM)
	}

	if sigErr != nil {
		return pid, sigErr
	}

	logger.InfoC("gateway", fmt.Sprintf("Sent stop signal to gateway (PID: %d)", pid))
	gateway.cmd = nil
	gateway.owned = false
	gateway.bootDefaultModel = ""
	gateway.pidData = nil
	// No current gateway means no valid idle credential, so a process that is
	// on its way out cannot trigger lifecycle work afterwards. PC-DEF-030.
	clearGatewayIdleToken()
	setGatewayRuntimeStatusLocked("stopped")

	return pid, nil
}

func stopGatewayProcessForRestart(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil || !isCmdProcessAliveLocked(cmd) {
		return nil
	}

	var stopErr error
	if runtime.GOOS == "windows" {
		stopErr = cmd.Process.Kill()
	} else {
		stopErr = cmd.Process.Signal(syscall.SIGTERM)
	}
	if stopErr != nil && isCmdProcessAliveLocked(cmd) {
		return fmt.Errorf("failed to stop existing gateway: %w", stopErr)
	}

	if waitForGatewayProcessExit(cmd, gatewayRestartGracePeriod) {
		return nil
	}

	if runtime.GOOS != "windows" {
		killErr := cmd.Process.Signal(syscall.SIGKILL)
		if killErr != nil && isCmdProcessAliveLocked(cmd) {
			return fmt.Errorf("failed to force-stop existing gateway: %w", killErr)
		}
		if waitForGatewayProcessExit(cmd, gatewayRestartForceKillWindow) {
			return nil
		}
	}

	return fmt.Errorf("existing gateway did not exit before restart")
}

func (h *Handler) startGatewayLocked(initialStatus string, existingPid int) (int, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return 0, fmt.Errorf("failed to load config: %w", err)
	}
	defaultModelName := strings.TrimSpace(cfg.Agents.Defaults.GetModelName())

	var cmd *exec.Cmd
	var pid int

	if existingPid > 0 {
		// Attach to existing process
		pid = existingPid
		gateway.cmd = nil // Clear first to ensure clean state
		if err = attachToGatewayProcessLocked(pid, cfg); err != nil {
			logger.ErrorC("gateway", fmt.Sprintf("Failed to attach to existing gateway (PID %d): %v", pid, err))
			return 0, err
		}

		return pid, nil
	}

	// Start new process
	// Locate the picoclaw executable
	execPath := utils.FindPicoclawBinary()
	logger.InfoC("gateway", "Starting gateway process")

	cmd = gatewayExecCommand(execPath, h.gatewayCommandArgs()...)
	applyLauncherProcAttrs(cmd)
	cmd.Env = os.Environ()
	// A fresh idle credential for this generation only, so a superseded
	// gateway cannot drive the lifecycle of the one that replaced it.
	if idleToken := newGatewayIdleToken(); idleToken != "" {
		cmd.Env = append(cmd.Env, config.EnvGatewayIdleToken+"="+idleToken)
		cmd.Env = append(cmd.Env, config.EnvGatewayIdleURL+"="+h.launcherIdleNotifyURL())
	}
	// Forward the launcher's config path via the environment variable that
	// GetConfigPath() already reads, so the gateway sub-process uses the same
	// config file without requiring a --config flag on the gateway subcommand.
	if h.configPath != "" {
		cmd.Env = append(cmd.Env, config.EnvConfig+"="+h.configPath)
	}
	gatewayHostOverride := h.gatewayHostOverride()
	if gatewayHostOverride != "" {
		cmd.Env = append(cmd.Env, config.EnvGatewayHost+"="+gatewayHostOverride)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Clear old logs for this new run
	gateway.logs.Reset()

	// Ensure the PocketClaw channel is configured before starting gateway
	changed, err := h.EnsurePocketClawChannel()
	if err != nil {
		logger.ErrorC("gateway", fmt.Sprintf("Warning: failed to ensure PocketClaw channel: %v", err))
		// Non-fatal: gateway can still start without the PocketClaw channel
	}
	// Refresh the cached PocketClaw token in case EnsurePocketClawChannel generated a new one.
	// Already holding gateway.mu from caller.
	if changed {
		refreshPocketClawTokensLocked(h.configPath)
		cfg, err = config.LoadConfig(h.configPath)
		if err != nil {
			return 0, fmt.Errorf("failed to reload config after ensuring the PocketClaw channel: %w", err)
		}
		defaultModelName = strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start gateway: %w", err)
	}

	gateway.cmd = cmd
	gateway.owned = true // We started this process
	gateway.bootDefaultModel = defaultModelName
	gateway.bootConfigSignature = computeConfigSignature(cfg)
	setGatewayRuntimeStatusLocked(initialStatus)
	pid = cmd.Process.Pid
	logger.InfoC("gateway", fmt.Sprintf("Started gateway (PID: %d)", pid))

	// Capture stdout/stderr in background. Both pipes must be drained before
	// cmd.Wait closes them, otherwise a child that dies during startup can lose
	// the very line that explains why.
	var pipesDone sync.WaitGroup
	pipesDone.Add(2)
	go func() {
		defer pipesDone.Done()
		scanPipe(stdoutPipe, gateway.logs)
	}()
	go func() {
		defer pipesDone.Done()
		scanPipe(stderrPipe, gateway.logs)
	}()

	startedAt := time.Now()

	// Wait for exit in background and clean up
	go func() {
		pipesDone.Wait()
		waitErr := cmd.Wait()
		if waitErr != nil {
			logger.ErrorC("gateway", fmt.Sprintf("Gateway process exited: %v", waitErr))
		} else {
			logger.InfoC("gateway", "Gateway process exited normally")
		}

		gateway.mu.Lock()
		if gateway.cmd == cmd {
			gateway.cmd = nil
			gateway.bootDefaultModel = ""
			gateway.bootConfigSignature = ""
			switch {
			case gateway.runtimeStatus == "restarting":
				// A restart owns the state machine; leave it to finish.
			case waitErr != nil && gateway.runtimeStatus == "starting":
				// Exited non-zero without ever reaching healthy. Land in a
				// terminal error state so the console stops saying "starting"
				// and the user can retry against a real message.
				lines, _, _ := gateway.logs.LinesSince(0)
				setGatewayRuntimeStatusLocked("error")
				gateway.lastStartupError = gatewayStartupFailureReason(lines, waitErr)
				logger.ErrorCF("gateway", "Gateway startup failed", map[string]any{
					"event":        "gateway.start.failed",
					"pid":          pid,
					"exit_error":   waitErr.Error(),
					"startup_ms":   time.Since(startedAt).Milliseconds(),
					"health_state": "never_healthy",
					"reason":       gateway.lastStartupError,
				})
			default:
				setGatewayRuntimeStatusLocked("stopped")
			}
		}
		gateway.mu.Unlock()
	}()

	// Start a goroutine to probe pidFile and health, update runtime state once ready.
	go func() {
		healthConfirmed := false
		for i := 0; i < 30; i++ { // try for up to 15 seconds
			time.Sleep(500 * time.Millisecond)
			gateway.mu.Lock()
			stillOurs := gateway.cmd == cmd
			gateway.mu.Unlock()
			if !stillOurs {
				return
			}

			// Poll for pidFile first — once available we have port/host/token.
			if pd := ppid.ReadPidFileWithCheck(globalConfigDir()); pd != nil && pd.PID == pid {
				gateway.mu.Lock()
				if gateway.cmd == cmd {
					gateway.pidData = pd
					var pocketClawCfg config.PocketClawSettings
					if bc := cfg.Channels.GetByType(config.ChannelPocketClaw); bc != nil {
						decoded, err := bc.GetDecoded()
						if err == nil && decoded != nil {
							if p, ok := decoded.(*config.PocketClawSettings); ok {
								pocketClawCfg = *p
							}
						}
					}
					gateway.pocketClawToken = pocketClawCfg.Token.String()
					setGatewayRuntimeStatusLocked("running")
				}
				gateway.mu.Unlock()
				logger.InfoC("gateway", fmt.Sprintf("Gateway pidFile detected (PID: %d, port: %d)", pd.PID, pd.Port))
				return
			}

			// Fallback: probe health endpoint to confirm liveness.
			cfg, err := config.LoadConfig(h.configPath)
			if err != nil {
				continue
			}
			_, statusCode, err := h.getGatewayHealth(cfg, 1*time.Second)
			if err == nil && statusCode == http.StatusOK {
				gateway.mu.Lock()
				if gateway.cmd == cmd {
					setGatewayRuntimeStatusLocked("running")
				}
				gateway.mu.Unlock()
				if !healthConfirmed {
					healthConfirmed = true
					logger.InfoC("gateway", "Gateway health endpoint reachable; waiting for pid file")
				}
				continue
			}
		}
	}()

	return pid, nil
}

// handleGatewayStart starts the picoclaw gateway subprocess.
//
//	POST /api/gateway/start
func (h *Handler) handleGatewayStart(w http.ResponseWriter, r *http.Request) {
	// Check PID file first to detect an already-running gateway.
	pidData := h.sanitizeGatewayPidData(ppid.ReadPidFileWithCheck(globalConfigDir()), nil, "manual_start")
	if pidData != nil {
		pid := pidData.PID
		gateway.mu.Lock()
		ready, reason, err := h.gatewayInfrastructureReady()
		if err != nil {
			gateway.mu.Unlock()
			http.Error(
				w,
				fmt.Sprintf("Failed to validate gateway start conditions: %v", err),
				http.StatusInternalServerError,
			)
			return
		}
		if !ready {
			gateway.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  "precondition_failed",
				"message": reason,
			})
			return
		}
		_, err = h.startGatewayLocked("starting", pid)
		if err != nil {
			gateway.mu.Unlock()
			logger.ErrorC("gateway", fmt.Sprintf("Failed to attach to running gateway (PID: %d): %v", pid, err))
			http.Error(w, fmt.Sprintf("Failed to attach to gateway: %v", err), http.StatusInternalServerError)
			return
		}
		gateway.pidData = pidData
		gateway.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"pid":    pid,
		})
		return
	}

	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	if gateway.cmd != nil && gateway.cmd.Process != nil {
		gateway.cmd = nil
		setGatewayRuntimeStatusLocked("stopped")
	}

	ready, reason, err := h.gatewayInfrastructureReady()
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Failed to validate gateway start conditions: %v", err),
			http.StatusInternalServerError,
		)
		return
	}
	if !ready {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "precondition_failed",
			"message": reason,
		})
		return
	}

	pid, err := h.startGatewayLocked("starting", 0)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start gateway: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"pid":    pid,
	})
}

// handleGatewayStop stops the running gateway subprocess gracefully.
// Note: Unlike StopGateway (which only stops self-started processes), this API endpoint
// stops any gateway process, including attached ones. This is intentional for user control.
//
//	POST /api/gateway/stop
func (h *Handler) handleGatewayStop(w http.ResponseWriter, r *http.Request) {
	gateway.mu.Lock()
	defer gateway.mu.Unlock()

	if gateway.cmd == nil || gateway.cmd.Process == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "not_running",
		})
		return
	}

	pid, err := stopGatewayLocked()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to stop gateway (PID %d): %v", pid, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"pid":    pid,
	})
}

// RestartGateway restarts the gateway process. This is a non-blocking operation
// that stops the current gateway (if running) and starts a new one.
// Returns the PID of the new gateway process or an error.
func (h *Handler) RestartGateway() (int, error) {
	// Adopt a live gateway this process is not tracking before deciding what to
	// stop. Without it the restart signals nothing, starts a second gateway
	// against a port the first one still holds, and leaves the old
	// configuration serving. PC-DEF-030.
	h.reconcileGatewayWithPidFile()

	ready, reason, err := h.gatewayInfrastructureReady()
	if err != nil {
		return 0, fmt.Errorf("failed to validate gateway start conditions: %w", err)
	}
	if !ready {
		return 0, &preconditionFailedError{reason: reason}
	}

	gateway.mu.Lock()
	previousCmd := gateway.cmd
	previousOwned := gateway.owned
	setGatewayRuntimeStatusLocked("restarting")
	gateway.mu.Unlock()

	if previousCmd != nil && previousCmd.Process != nil && !previousOwned {
		if isGateway, inspected := gatewayProcessMatcher(previousCmd.Process.Pid); inspected && !isGateway {
			logger.Warnf("refuse restarting non-gateway process (PID: %d)", previousCmd.Process.Pid)
			gateway.mu.Lock()
			if gateway.cmd == previousCmd {
				setGatewayRuntimeStatusLocked("running")
			}
			gateway.mu.Unlock()
			return 0, fmt.Errorf("refuse to restart non-gateway process (PID %d)", previousCmd.Process.Pid)
		}
	}

	if err = stopGatewayProcessForRestart(previousCmd); err != nil {
		gateway.mu.Lock()
		if gateway.cmd == previousCmd {
			if isCmdProcessAliveLocked(previousCmd) {
				setGatewayRuntimeStatusLocked("running")
			} else {
				gateway.cmd = nil
				gateway.bootDefaultModel = ""
				setGatewayRuntimeStatusLocked("error")
			}
		}
		gateway.mu.Unlock()
		return 0, fmt.Errorf("failed to stop gateway: %w", err)
	}

	gateway.mu.Lock()
	if gateway.cmd == previousCmd {
		gateway.cmd = nil
		gateway.bootDefaultModel = ""
	}
	pid, err := h.startGatewayLocked("restarting", 0)
	if err != nil {
		gateway.cmd = nil
		gateway.bootDefaultModel = ""
		setGatewayRuntimeStatusLocked("error")
	}
	gateway.mu.Unlock()
	if err != nil {
		return 0, fmt.Errorf("failed to start gateway: %w", err)
	}

	return pid, nil
}

// preconditionFailedError is returned when gateway restart preconditions are not met
type preconditionFailedError struct {
	reason string
}

func (e *preconditionFailedError) Error() string {
	return e.reason
}

// IsBadRequest returns true if the error should result in a 400 Bad Request status
func (e *preconditionFailedError) IsBadRequest() bool {
	return true
}

// handleGatewayRestart stops the gateway (if running) and starts a new instance.
//
//	POST /api/gateway/restart
func (h *Handler) handleGatewayRestart(w http.ResponseWriter, r *http.Request) {
	pid, err := h.RestartGateway()
	if err != nil {
		// Check if it's a precondition failed error
		var precondErr *preconditionFailedError
		if errors.As(err, &precondErr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"status":  "precondition_failed",
				"message": precondErr.reason,
			})
			return
		}
		http.Error(w, fmt.Sprintf("Failed to restart gateway: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"pid":    pid,
	})
}

// handleGatewayClearLogs clears the in-memory gateway log buffer.
//
//	POST /api/gateway/logs/clear
func (h *Handler) handleGatewayClearLogs(w http.ResponseWriter, r *http.Request) {
	gateway.logs.Clear()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "cleared",
		"log_total":  0,
		"log_run_id": gateway.logs.RunID(),
	})
}

// handleGatewayStatus returns the gateway run status and health info.
//
//	GET /api/gateway/status
func (h *Handler) handleGatewayStatus(w http.ResponseWriter, r *http.Request) {
	data := h.gatewayStatusData()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) gatewayStatusData() map[string]any {
	data := map[string]any{}
	var configDefaultModel string
	cfg, cfgErr := config.LoadConfig(h.configPath)
	if cfgErr == nil && cfg != nil {
		configDefaultModel = strings.TrimSpace(cfg.Agents.Defaults.GetModelName())
		if configDefaultModel != "" {
			data["config_default_model"] = configDefaultModel
		}
	}

	// Primary detection: read PID file and check if process is alive.
	pidData := h.sanitizeGatewayPidData(ppid.ReadPidFileWithCheck(globalConfigDir()), cfg, "status")
	if pidData != nil {
		gateway.mu.Lock()
		gateway.pidData = pidData
		if pidData.Version != "" {
			data["gateway_version"] = pidData.Version
		}
		setGatewayRuntimeStatusLocked("running")

		// Attach if we don't already track this PID.
		if gateway.cmd == nil || gateway.cmd.Process == nil || gateway.cmd.Process.Pid != pidData.PID {
			_ = attachToGatewayProcessLocked(pidData.PID, cfg)
		}

		bootDefaultModel := gateway.bootDefaultModel
		if bootDefaultModel != "" {
			data["boot_default_model"] = bootDefaultModel
		}
		data["gateway_status"] = "running"
		data["pid"] = pidData.PID
		gateway.mu.Unlock()
	} else {
		// Intentionally skip health probe here; the startup goroutine
		// (startGatewayLocked) already handles liveness detection via
		// pidFile polling and health fallback.
		gateway.mu.Lock()
		status := gatewayStatusWithoutHealthLocked()
		data["gateway_status"] = status
		if status == "error" && gateway.lastStartupError != "" {
			data["gateway_last_error"] = gateway.lastStartupError
		}
		// Keep last known pidData while gateway is still in a transient
		// running state; otherwise websocket proxy may lose auth token
		// during short pid-file races.
		if status == "stopped" || status == "error" {
			gateway.pidData = nil
		}
		gateway.mu.Unlock()
	}

	gatewayStatus, _ := data["gateway_status"].(string)
	currentConfigSignature := computeConfigSignature(cfg)
	gateway.mu.Lock()
	bootConfigSignature := gateway.bootConfigSignature
	gateway.mu.Unlock()
	data["gateway_restart_required"] = gatewayRestartRequiredBySignature(
		bootConfigSignature,
		currentConfigSignature,
		gatewayStatus,
	)

	// Gateway health and AI availability are independent facts and are
	// reported as such: "Gateway: Running / AI provider: Not configured" is a
	// valid, expected state, not a degraded one. See PC-DEF-038.
	ready, reason, readyErr := h.gatewayInfrastructureReady()
	if readyErr != nil {
		data["gateway_start_allowed"] = false
		data["gateway_start_reason"] = readyErr.Error()
	} else {
		data["gateway_start_allowed"] = ready
		if !ready {
			data["gateway_start_reason"] = reason
		}
	}

	// Read-only by contract. Monitoring must never apply, restart, clear or
	// retry anything -- the gateway's idle notification is the only trigger.
	applyPending, applyErr := pendingConfigApplyState()
	data["config_apply_pending"] = applyPending
	if applyErr != "" {
		data["config_apply_error"] = applyErr
	}

	chatOK, chatReason, chatErr := h.chatReady()
	if chatErr != nil {
		data["chat_ready"] = false
		data["chat_not_ready_reason"] = chatErr.Error()
	} else {
		data["chat_ready"] = chatOK
		if !chatOK {
			data["chat_not_ready_reason"] = chatReason
		}
	}

	return data
}

// handleGatewayLogs returns buffered gateway logs, optionally incrementally.
//
//	GET /api/gateway/logs
func (h *Handler) handleGatewayLogs(w http.ResponseWriter, r *http.Request) {
	data := gatewayLogsData(r)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// gatewayLogsData reads log_offset and log_run_id query params from the request
// and returns incremental log lines.
func gatewayLogsData(r *http.Request) map[string]any {
	data := map[string]any{}
	clientOffset := 0
	clientRunID := -1

	if v := r.URL.Query().Get("log_offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			clientOffset = n
		}
	}

	if v := r.URL.Query().Get("log_run_id"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			clientRunID = n
		}
	}

	runID := gateway.logs.RunID()

	if runID == 0 {
		data["logs"] = []string{}
		data["log_total"] = 0
		data["log_run_id"] = 0
		return data
	}

	// If runID changed, reset offset to get all logs from new run
	offset := clientOffset
	if clientRunID != runID {
		offset = 0
	}

	lines, total, runID := gateway.logs.LinesSince(offset)
	if lines == nil {
		lines = []string{}
	}

	data["logs"] = lines
	data["log_total"] = total
	data["log_run_id"] = runID
	return data
}

// scanPipe reads lines from r and appends them to buf. Returns when r reaches EOF.
// gatewayStartupFailurePatterns maps a marker in the child's captured output to
// the one sentence the console should show. The list is a whitelist on purpose:
// the gateway's own logs carry config paths and channel detail, and a panic
// dump can carry anything at all, so no child output is ever forwarded
// verbatim. Order matters — the first match wins.
var gatewayStartupFailurePatterns = []struct {
	marker string
	reason string
}{
	{"singleton check failed", "Gateway could not start because a stale previous process record was detected."},
	{"gateway is already running", "Gateway could not start because a stale previous process record was detected."},
	{"error opening gateway listeners", "Gateway could not start because its network port is already in use."},
	{"config pre-check failed", "Gateway could not start because its configuration was rejected."},
	{"error loading config", "Gateway could not start because its configuration could not be read."},
	{"error enabling file logging", "Gateway could not start because it could not open its log file."},
}

// gatewayStartupFailureReason turns a failed child's captured output into one
// short, safe sentence. Unrecognised failures fall back to the exit status
// alone, which carries no configuration or credential data.
func gatewayStartupFailureReason(lines []string, waitErr error) string {
	for i := len(lines) - 1; i >= 0; i-- {
		lower := strings.ToLower(lines[i])
		for _, p := range gatewayStartupFailurePatterns {
			if strings.Contains(lower, p.marker) {
				return p.reason
			}
		}
	}
	if waitErr != nil {
		return fmt.Sprintf("Gateway exited during startup (%v). Check the gateway logs for details.", waitErr)
	}
	return "Gateway exited during startup. Check the gateway logs for details."
}

func scanPipe(r io.Reader, buf *LogBuffer) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		buf.Append(scanner.Text())
	}
}
