package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// LauncherDashboardCookieName is the HttpOnly cookie set after a successful password login.
const LauncherDashboardCookieName = "pocketclaw_launcher_auth"

// legacyLauncherDashboardCookieName is LEGACY READ-ONLY SESSION HANDOFF.
//
// It is never issued and never consulted for authentication. It is named here
// only so a cookie left in the browser by an older build can be expired.
//
// There is deliberately no session handoff from it. Sessions live in
// LauncherDashboardSessions, which is an in-memory map created fresh at process
// start: the process that issued a legacy cookie is by definition gone, so no
// legacy value can name a live session. Accepting one would mean trusting a
// bearer token with no server-side record — the exact bypass this cookie exists
// to prevent — and an upgrade already costs one dashboard login, because the
// session store has never survived a restart under either name.
const legacyLauncherDashboardCookieName = "picoclaw_launcher_auth"

// launcherDashboardSessionMaxAgeSec is the dashboard session cookie lifetime.
const launcherDashboardSessionMaxAgeSec = 24 * 3600

const (
	launcherSessionCookieBytes = 32
	launcherGrantNonceBytes    = 32
	// LauncherDashboardLocalAutoLoginPath is the one-shot local browser
	// bootstrap endpoint used by the launcher-managed auto-open flow.
	LauncherDashboardLocalAutoLoginPath = "/launcher-auto-login"
	// LauncherDashboardSetupPath is the setup page used before the dashboard
	// password is initialized.
	LauncherDashboardSetupPath = "/launcher-setup"
)

// NewLauncherDashboardSessionCookie creates the per-process session cookie value.
func NewLauncherDashboardSessionCookie() (string, error) {
	return randomURLToken(launcherSessionCookieBytes)
}

func randomURLToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// LauncherDashboardAuthConfig holds runtime material for dashboard access checks.
type LauncherDashboardAuthConfig struct {
	ExpectedCookie string
	// Sessions is the authoritative server-side dashboard session store. When
	// set, cookies are accepted only while their corresponding session remains
	// live and has not been revoked.
	Sessions *LauncherDashboardSessions
	// LocalAutoLogin enables one-shot startup auto-login.
	LocalAutoLogin *LauncherDashboardLocalAutoLogin
	// SecureCookie sets the session cookie's Secure flag. If nil, DefaultLauncherDashboardSecureCookie is used.
	SecureCookie func(*http.Request) bool
}

// LauncherDashboardSessions tracks short-lived dashboard sessions in memory.
// Session tokens are opaque, cryptographically random bearer values; only the
// browser cookie contains the value and logs must never include it.
type LauncherDashboardSessions struct {
	mu       sync.Mutex
	ttl      time.Duration
	sessions map[string]time.Time
	now      func() time.Time
}

// NewLauncherDashboardSessions creates an empty server-side session store.
func NewLauncherDashboardSessions(ttl time.Duration) *LauncherDashboardSessions {
	if ttl <= 0 {
		ttl = time.Duration(launcherDashboardSessionMaxAgeSec) * time.Second
	}
	return &LauncherDashboardSessions{
		ttl:      ttl,
		sessions: make(map[string]time.Time),
		now:      time.Now,
	}
}

// Issue creates and records a fresh dashboard session.
func (s *LauncherDashboardSessions) Issue() (string, error) {
	if s == nil {
		return "", errors.New("dashboard session store unavailable")
	}
	token, err := NewLauncherDashboardSessionCookie()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = s.currentTime().Add(s.ttl)
	return token, nil
}

// Valid reports whether token names a live, non-revoked session.
func (s *LauncherDashboardSessions) Valid(token string) bool {
	if s == nil || token == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	expires, ok := s.sessions[token]
	if !ok {
		return false
	}
	if !s.currentTime().Before(expires) {
		delete(s.sessions, token)
		return false
	}
	return true
}

// Revoke invalidates a session immediately.
func (s *LauncherDashboardSessions) Revoke(token string) {
	if s == nil || token == "" {
		return
	}
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *LauncherDashboardSessions) currentTime() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// LauncherDashboardLocalAutoLogin is an in-memory, one-shot startup grant.
// It is not a reusable credential; it only lets the launcher-opened browser
// receive the current process session cookie.
type LauncherDashboardLocalAutoLogin struct {
	grant *launcherDashboardOneTimeGrant
}

type launcherDashboardOneTimeGrant struct {
	mu       sync.Mutex
	expires  time.Time
	consumed bool
	nonce    string
	now      func() time.Time
}

// NewLauncherDashboardLocalAutoLogin creates a one-shot local auto-login grant.
func NewLauncherDashboardLocalAutoLogin(ttl time.Duration) (*LauncherDashboardLocalAutoLogin, error) {
	grant, err := newLauncherDashboardOneTimeGrant(ttl)
	if err != nil {
		return nil, err
	}
	return &LauncherDashboardLocalAutoLogin{
		grant: grant,
	}, nil
}

// URLPath returns the one-shot local auto-login URL path including its nonce.
func (a *LauncherDashboardLocalAutoLogin) URLPath() string {
	return launcherGrantQueryPath(LauncherDashboardLocalAutoLoginPath, a.grant)
}

// DefaultLauncherDashboardSecureCookie mirrors typical production HTTPS detection (TLS or X-Forwarded-Proto).
func DefaultLauncherDashboardSecureCookie(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// SetLauncherDashboardSessionCookie writes the HttpOnly session cookie after successful dashboard password login.
func SetLauncherDashboardSessionCookie(
	w http.ResponseWriter,
	r *http.Request,
	sessionValue string,
	secure func(*http.Request) bool,
) {
	if secure == nil {
		secure = DefaultLauncherDashboardSecureCookie
	}
	http.SetCookie(w, &http.Cookie{
		Name:     LauncherDashboardCookieName,
		Value:    sessionValue,
		Path:     "/",
		MaxAge:   launcherDashboardSessionMaxAgeSec,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure(r),
	})
	// Login is the moment the browser is known to be presenting whatever it
	// still holds, so it is where a cookie from an older build is retired.
	expireLegacyLauncherDashboardSessionCookie(w, r, secure)
}

// expireLegacyLauncherDashboardSessionCookie removes a cookie left by a build
// from before the name migration.
//
// Path and the security attributes match what that build set, because a
// deletion whose Path does not match the original leaves the cookie in place
// and silently does nothing.
func expireLegacyLauncherDashboardSessionCookie(
	w http.ResponseWriter,
	r *http.Request,
	secure func(*http.Request) bool,
) {
	if secure == nil {
		secure = DefaultLauncherDashboardSecureCookie
	}
	http.SetCookie(w, &http.Cookie{
		Name:     legacyLauncherDashboardCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure(r),
		Expires:  time.Unix(0, 0),
	})
}

// ClearLauncherDashboardSessionCookie clears the dashboard session (e.g. logout).
func ClearLauncherDashboardSessionCookie(w http.ResponseWriter, r *http.Request, secure func(*http.Request) bool) {
	if secure == nil {
		secure = DefaultLauncherDashboardSecureCookie
	}
	http.SetCookie(w, &http.Cookie{
		Name:     LauncherDashboardCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure(r),
		Expires:  time.Unix(0, 0),
	})
	// Logout retires both names, so a browser that still carries the old one
	// does not keep an inert cookie for the rest of its life.
	expireLegacyLauncherDashboardSessionCookie(w, r, secure)
}

// LauncherDashboardAuth requires a valid session cookie before calling next.
// Public paths are login/setup pages and /api/auth/* handlers.
func LauncherDashboardAuth(cfg LauncherDashboardAuthConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := canonicalAuthPath(r.URL.Path)
		if p == LauncherDashboardLocalAutoLoginPath {
			handleLauncherLocalAutoLogin(w, r, cfg)
			return
		}
		if isPublicLauncherDashboardPath(r.Method, p) {
			next.ServeHTTP(w, r)
			return
		}
		if validLauncherDashboardAuth(r, cfg) {
			if p == config.RealtimeWebSocketPath && !validLauncherWebSocketOrigin(r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		rejectLauncherDashboardAuth(w, r, p)
	})
}

// canonicalAuthPath matches path cleaning used for routing decisions so
// prefixes like /assets/../ cannot bypass auth (CVE-class traversal).

func handleLauncherLocalAutoLogin(w http.ResponseWriter, r *http.Request, cfg LauncherDashboardAuthConfig) {
	if validLauncherDashboardAuth(r, cfg) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte("method not allowed"))
		return
	}
	if r.Method == http.MethodHead {
		rejectLauncherDashboardAuth(w, r, LauncherDashboardLocalAutoLoginPath)
		return
	}
	if cfg.LocalAutoLogin != nil && cfg.LocalAutoLogin.consume(r.URL.Query().Get("nonce")) {
		sessionValue := cfg.ExpectedCookie
		if cfg.Sessions != nil {
			var err error
			sessionValue, err = cfg.Sessions.Issue()
			if err != nil {
				http.Error(w, "dashboard session unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		SetLauncherDashboardSessionCookie(w, r, sessionValue, cfg.SecureCookie)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	rejectLauncherDashboardAuth(w, r, LauncherDashboardLocalAutoLoginPath)
}

func (a *LauncherDashboardLocalAutoLogin) consume(nonce string) bool {
	if a == nil || a.grant == nil {
		return false
	}
	return a.grant.use(nonce, nil) == nil
}

func newLauncherDashboardOneTimeGrant(ttl time.Duration) (*launcherDashboardOneTimeGrant, error) {
	nonce, err := randomURLToken(launcherGrantNonceBytes)
	if err != nil {
		return nil, err
	}
	return &launcherDashboardOneTimeGrant{
		expires: time.Now().Add(ttl),
		nonce:   nonce,
		now:     time.Now,
	}, nil
}

func launcherGrantQueryPath(basePath string, grant *launcherDashboardOneTimeGrant) string {
	if grant == nil {
		return basePath
	}
	return basePath + "?nonce=" + url.QueryEscape(grant.nonce)
}

// ErrInvalidLauncherDashboardGrant reports that an auto-login grant is missing,
// expired, already consumed, or otherwise invalid.
var ErrInvalidLauncherDashboardGrant = errors.New("invalid launcher dashboard grant")

func (g *launcherDashboardOneTimeGrant) use(nonce string, fn func() error) error {
	if g == nil {
		return ErrInvalidLauncherDashboardGrant
	}
	if len(nonce) != len(g.nonce) ||
		subtle.ConstantTimeCompare([]byte(nonce), []byte(g.nonce)) != 1 {
		return ErrInvalidLauncherDashboardGrant
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now
	if g.now != nil {
		now = g.now
	}
	if g.consumed || !now().Before(g.expires) {
		return ErrInvalidLauncherDashboardGrant
	}
	if fn != nil {
		if err := fn(); err != nil {
			return err
		}
	}
	g.consumed = true
	return nil
}

func canonicalAuthPath(raw string) string {
	if raw == "" {
		return "/"
	}
	c := path.Clean(raw)
	switch c {
	case ".", "":
		return "/"
	default:
		if c[0] != '/' {
			return "/" + c
		}
		return c
	}
}

func isPublicLauncherDashboardPath(method, p string) bool {
	if isPublicLauncherDashboardStatic(method, p) {
		return true
	}
	switch p {
	case "/api/auth/login":
		return method == http.MethodPost
	case "/api/auth/logout":
		return method == http.MethodPost
	case "/api/auth/status":
		return method == http.MethodGet
	case "/api/auth/setup":
		return method == http.MethodPost
	}
	return false
}

// isPublicLauncherDashboardStatic allows the SPA login route and embedded
// frontend assets without a session (GET/HEAD only).
func isPublicLauncherDashboardStatic(method, p string) bool {
	if method != http.MethodGet && method != http.MethodHead {
		return false
	}
	if p == "/launcher-login" || p == "/launcher-setup" {
		return true
	}
	if strings.HasPrefix(p, "/assets/") {
		return true
	}
	switch p {
	case "/favicon.ico", "/favicon.svg", "/favicon-96x96.png",
		"/apple-touch-icon.png", "/site.webmanifest", "/robots.txt":
		return true
	default:
		return false
	}
}

// validLauncherDashboardAuth reports whether the request carries a live session.
//
// The canonical cookie only. A legacy cookie is never a fallback: falling back
// when the canonical one is absent would be pointless (no legacy value can name
// a live in-memory session), and falling back when it is present but invalid
// would let an old cookie rescue a rejected session, which is an auth bypass.
func validLauncherDashboardAuth(r *http.Request, cfg LauncherDashboardAuthConfig) bool {
	if c, err := r.Cookie(LauncherDashboardCookieName); err == nil {
		if cfg.Sessions != nil {
			return cfg.Sessions.Valid(c.Value)
		}
		if subtle.ConstantTimeCompare([]byte(c.Value), []byte(cfg.ExpectedCookie)) == 1 {
			return true
		}
	}
	return false
}

func validLauncherWebSocketOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	expectedHost := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	if expectedHost == "" {
		expectedHost = strings.TrimSpace(r.Host)
	}
	return expectedHost != "" && strings.EqualFold(u.Host, expectedHost)
}

func rejectLauncherDashboardAuth(w http.ResponseWriter, r *http.Request, canonicalPath string) {
	if canonicalPath == config.RealtimeWebSocketPath {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if strings.HasPrefix(canonicalPath, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
		return
	}
	http.Redirect(w, r, "/launcher-login", http.StatusFound)
}
