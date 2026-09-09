package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

const legacyCookieName = "picoclaw_launcher_auth"

func cookieNamed(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestLoginIssuesOnlyTheCanonicalCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	SetLauncherDashboardSessionCookie(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", nil),
		"session-value", func(*http.Request) bool { return false })

	cookies := rec.Result().Cookies()
	session := cookieNamed(cookies, LauncherDashboardCookieName)
	if session == nil || session.Value != "session-value" {
		t.Fatalf("no canonical session cookie: %#v", cookies)
	}
	legacy := cookieNamed(cookies, legacyCookieName)
	if legacy == nil {
		t.Fatal("the legacy cookie should be expired on login")
	}
	if legacy.Value != "" || legacy.MaxAge >= 0 {
		t.Errorf("the legacy cookie was issued rather than expired: %#v", legacy)
	}
}

// The attributes are the security contract, not decoration: HttpOnly keeps the
// value away from JS, SameSite=Lax blocks cross-site submission, Path=/ scopes
// it to the dashboard and MaxAge holds the 24h session lifetime.
func TestCanonicalCookieKeepsTheSecurityAttributes(t *testing.T) {
	for _, secure := range []bool{false, true} {
		rec := httptest.NewRecorder()
		SetLauncherDashboardSessionCookie(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", nil),
			"session-value", func(*http.Request) bool { return secure })

		session := cookieNamed(rec.Result().Cookies(), LauncherDashboardCookieName)
		if session == nil {
			t.Fatal("no session cookie")
		}
		if !session.HttpOnly {
			t.Error("session cookie is not HttpOnly")
		}
		if session.SameSite != http.SameSiteLaxMode {
			t.Errorf("SameSite = %v, want Lax", session.SameSite)
		}
		if session.Path != "/" {
			t.Errorf("Path = %q, want /", session.Path)
		}
		if session.MaxAge != 24*3600 {
			t.Errorf("MaxAge = %d, want 24 hours", session.MaxAge)
		}
		if session.Secure != secure {
			t.Errorf("Secure = %v, want %v", session.Secure, secure)
		}
		if session.Domain != "" {
			t.Errorf("Domain = %q, want host-only", session.Domain)
		}
	}
}

// A deletion whose Path does not match the original leaves the cookie in place
// and silently does nothing, which is the failure mode worth pinning.
func TestLegacyExpiryUsesAttributesThatActuallyRemoveIt(t *testing.T) {
	rec := httptest.NewRecorder()
	SetLauncherDashboardSessionCookie(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", nil),
		"session-value", func(*http.Request) bool { return true })

	legacy := cookieNamed(rec.Result().Cookies(), legacyCookieName)
	if legacy == nil {
		t.Fatal("no legacy expiry")
	}
	if legacy.Path != "/" {
		t.Errorf("Path = %q, want / to match what the old build set", legacy.Path)
	}
	if !legacy.HttpOnly || legacy.SameSite != http.SameSiteLaxMode {
		t.Errorf("legacy expiry does not match the original attributes: %#v", legacy)
	}
	if legacy.MaxAge >= 0 || !legacy.Expires.Equal(time.Unix(0, 0)) {
		t.Errorf("legacy cookie is not being expired: %#v", legacy)
	}
}

func TestLogoutRetiresBothNames(t *testing.T) {
	rec := httptest.NewRecorder()
	ClearLauncherDashboardSessionCookie(rec, httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil),
		func(*http.Request) bool { return false })

	for _, name := range []string{LauncherDashboardCookieName, legacyCookieName} {
		c := cookieNamed(rec.Result().Cookies(), name)
		if c == nil {
			t.Errorf("logout did not clear %q", name)
			continue
		}
		if c.Value != "" || c.MaxAge >= 0 {
			t.Errorf("logout did not expire %q: %#v", name, c)
		}
	}
}

// The whole state space that can exist. A legacy cookie is never valid — the
// in-memory session store that could validate it died with the process that
// issued it — so what matters is that it never authenticates and never rescues
// a rejected canonical session.
func TestLegacyCookieNeverAuthenticates(t *testing.T) {
	sessions := NewLauncherDashboardSessions(time.Hour)
	live, err := sessions.Issue()
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := sessions.Issue()
	if err != nil {
		t.Fatal(err)
	}
	sessions.Revoke(revoked)

	cases := []struct {
		name      string
		canonical string
		legacy    string
		want      bool
	}{
		{name: "canonical only", canonical: live, want: true},
		{name: "no cookies at all", want: false},
		{name: "legacy holding a live token, canonical absent", legacy: live, want: false},
		{name: "legacy holding a revoked token, canonical absent", legacy: revoked, want: false},
		{name: "both valid: canonical is the one that counts", canonical: live, legacy: live, want: true},
		{name: "valid canonical beside a junk legacy", canonical: live, legacy: "junk", want: true},
		// The bypass this ordering exists to prevent.
		{name: "revoked canonical is not rescued by a live legacy", canonical: revoked, legacy: live, want: false},
		{name: "junk canonical is not rescued by a live legacy", canonical: "junk", legacy: live, want: false},
		{name: "both junk", canonical: "junk", legacy: "junk", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.canonical != "" {
				req.AddCookie(&http.Cookie{Name: LauncherDashboardCookieName, Value: tc.canonical})
			}
			if tc.legacy != "" {
				req.AddCookie(&http.Cookie{Name: legacyCookieName, Value: tc.legacy})
			}
			got := validLauncherDashboardAuth(req, LauncherDashboardAuthConfig{Sessions: sessions})
			if got != tc.want {
				t.Fatalf("validLauncherDashboardAuth = %v, want %v", got, tc.want)
			}
		})
	}
}

// Reissuing under a new name must not become a free session extension, and the
// server-side store stays authoritative.
func TestCookieMigrationDoesNotTouchServerSideSessions(t *testing.T) {
	sessions := NewLauncherDashboardSessions(time.Hour)
	token, err := sessions.Issue()
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: legacyCookieName, Value: token})

	if validLauncherDashboardAuth(req, LauncherDashboardAuthConfig{Sessions: sessions}) {
		t.Fatal("a legacy cookie authenticated")
	}
	// No canonical cookie was minted from it, and nothing was written back.
	if len(rec.Result().Cookies()) != 0 {
		t.Errorf("the auth check issued cookies: %#v", rec.Result().Cookies())
	}

	sessions.Revoke(token)
	if sessions.Valid(token) {
		t.Error("revocation no longer ends the session")
	}
}

// Nothing in production may set the legacy name to a value.
func TestNoProductionCodeIssuesTheLegacyCookie(t *testing.T) {
	root := moduleRoot(t)
	// The one file allowed to name it, and only to expire it.
	const owner = "web/backend/middleware/launcher_dashboard_auth.go"
	legacy := regexp.MustCompile(`"picoclaw_launcher_auth"`)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case "vendor", "build", "node_modules", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if filepath.ToSlash(relative) == owner {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if legacy.Match(raw) {
			t.Errorf("%s names the legacy session cookie", filepath.ToSlash(relative))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// And in the owner it appears exactly once: the constant.
	raw, err := os.ReadFile(filepath.Join(root, owner))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(legacy.FindAll(raw, -1)); got != 1 {
		t.Errorf("the legacy cookie literal appears %d times in the migration owner, want 1", got)
	}
	if !strings.Contains(string(raw), "LEGACY READ-ONLY SESSION HANDOFF") {
		t.Error("the legacy constant is not classified")
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
