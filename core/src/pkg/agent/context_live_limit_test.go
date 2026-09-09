package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// writeLimitConfig persists a config the way the Settings bridge does — through
// Core's own SaveConfig — so the test exercises the real write path rather than
// a hand-built file.
func writeLimitConfig(t *testing.T, path string, limit int) {
	t.Helper()
	cfg, err := config.LoadConfig(path)
	if err != nil {
		cfg = config.DefaultConfig()
	}
	cfg.Agents.Defaults.TelegramRecentContextMessages = limit
	if err := config.SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig(%d): %v", limit, err)
	}
	// Some filesystems report whole-second mtimes; make a change detectable
	// without depending on timer resolution.
	future := time.Now().Add(time.Duration(limit) * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
}

// liveLimitFixture points the resolver at a temporary config and restores the
// process-wide cache afterwards.
func liveLimitFixture(t *testing.T, initial int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	writeLimitConfig(t, path, initial)

	previous := telegramContextLimit
	telegramContextLimit = &liveTelegramContextLimit{path: path}
	t.Cleanup(func() { telegramContextLimit = previous })
	return path
}

// telegramAgent is a running agent that started before any of these saves.
// It is deliberately never rebuilt: the whole point is that a live change
// reaches it without reconstruction.
func telegramAgent(startupValue int) *AgentInstance {
	return &AgentInstance{TelegramRecentContextMessages: startupValue}
}

// THE PHYSICAL FAILURE. A saved 17 kept projecting 15 until a restart.
func TestSavedLimitAppliesToTheNextTurnWithoutRestart(t *testing.T) {
	path := liveLimitFixture(t, 15)
	agent := telegramAgent(15)

	if got := recentContextLimit(agent, "telegram"); got != 15 {
		t.Fatalf("initial limit = %d, want 15", got)
	}

	writeLimitConfig(t, path, 17)
	if got := recentContextLimit(agent, "telegram"); got != 17 {
		t.Fatalf("after saving 17 the limit was %d; the running agent did not observe the change", got)
	}

	writeLimitConfig(t, path, 10)
	if got := recentContextLimit(agent, "telegram"); got != 10 {
		t.Fatalf("after saving 10 the limit was %d, want 10", got)
	}

	// The agent object was never rebuilt and still holds its startup value.
	if agent.TelegramRecentContextMessages != 15 {
		t.Fatal("the test rebuilt or mutated the agent; it must observe the file instead")
	}
}

// The projection itself must use the live value, not just the resolver.
func TestProjectionUsesTheLiveLimit(t *testing.T) {
	path := liveLimitFixture(t, 15)
	ts := &turnState{
		agent:      telegramAgent(15),
		sessionKey: "s1",
		opts:       processOptions{Channel: "telegram"},
	}

	history := conversation(40)

	writeLimitConfig(t, path, 30)
	wide := ts.projectRecentContext(context.Background(), nil, history, "")

	writeLimitConfig(t, path, 6)
	narrow := ts.projectRecentContext(context.Background(), nil, history, "")

	if len(narrow) >= len(wide) {
		t.Fatalf("a smaller saved limit did not narrow the projection: %d then %d",
			len(wide), len(narrow))
	}
}

// An invalid or out-of-range stored value must not move the effective limit.
func TestInvalidStoredValueLeavesTheEffectiveLimitAtTheDefault(t *testing.T) {
	path := liveLimitFixture(t, 20)
	agent := telegramAgent(15)

	if got := recentContextLimit(agent, "telegram"); got != 20 {
		t.Fatalf("limit = %d, want the stored 20", got)
	}

	for _, invalid := range []int{0, -5, 4, 51, 100000} {
		writeLimitConfig(t, path, invalid)
		if got := recentContextLimit(agent, "telegram"); got != config.DefaultTelegramRecentContextMessages {
			t.Fatalf("stored %d resolved to %d, want the default %d",
				invalid, got, config.DefaultTelegramRecentContextMessages)
		}
	}
}

// A save that never landed must not move the runtime value.
func TestFailedPersistenceLeavesTheRuntimeValueUnchanged(t *testing.T) {
	path := liveLimitFixture(t, 20)
	agent := telegramAgent(15)
	if got := recentContextLimit(agent, "telegram"); got != 20 {
		t.Fatalf("limit = %d, want 20", got)
	}

	// The file is unchanged because the write failed. Nothing may shift.
	for i := 0; i < 3; i++ {
		if got := recentContextLimit(agent, "telegram"); got != 20 {
			t.Fatalf("limit drifted to %d with no successful save", got)
		}
	}

	// A truncated file — a save caught mid-write — keeps the last good answer
	// rather than snapping to the default.
	if err := os.WriteFile(path, []byte("{ this is not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := recentContextLimit(agent, "telegram"); got != 20 {
		t.Fatalf("a malformed config changed the limit to %d, want the previous 20", got)
	}
}

// An unreadable config falls back to what the agent started with.
func TestMissingConfigFallsBackToTheStartupValue(t *testing.T) {
	path := liveLimitFixture(t, 20)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if got := recentContextLimit(telegramAgent(15), "telegram"); got != 15 {
		t.Fatalf("limit = %d with no config file, want the startup 15", got)
	}
}

// Only Telegram consults the live limit; every other channel stays unbounded.
func TestLiveLimitIsStillTelegramOnly(t *testing.T) {
	liveLimitFixture(t, 25)
	agent := telegramAgent(15)
	for _, channel := range []string{"pocketclaw", "discord", "slack", "matrix", "cli"} {
		if got := recentContextLimit(agent, channel); got != 0 {
			t.Errorf("%s limit = %d, want 0", channel, got)
		}
	}
}

// An unchanged file must not be re-decoded on every projection.
func TestUnchangedConfigIsNotRereadEveryTime(t *testing.T) {
	path := liveLimitFixture(t, 20)
	agent := telegramAgent(15)

	if got := recentContextLimit(agent, "telegram"); got != 20 {
		t.Fatalf("limit = %d, want 20", got)
	}

	// Replace the file contents without touching size or mtime: a re-decode
	// would notice, a correctly cached read must not.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := make([]byte, len(original))
	copy(tampered, original)
	if err := os.WriteFile(path, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	if got := recentContextLimit(agent, "telegram"); got != 20 {
		t.Fatalf("limit = %d after an identical rewrite, want the cached 20", got)
	}
}

// The limit change must not disturb stored conversation state.
func TestLimitChangeLeavesHistoryAndSummaryAlone(t *testing.T) {
	path := liveLimitFixture(t, 15)
	ts := &turnState{
		agent:      telegramAgent(15),
		sessionKey: "s1",
		opts:       processOptions{Channel: "telegram"},
	}

	history := conversation(12)
	before := make([]providers.Message, len(history))
	copy(before, history)
	const summary = "a persisted rolling summary with EXACT FACTS: NEBULA-9634"

	writeLimitConfig(t, path, 25)
	ts.projectRecentContext(context.Background(), nil, history, summary)

	if len(history) != len(before) {
		t.Fatalf("history length changed from %d to %d", len(before), len(history))
	}
	for i := range before {
		if history[i].Content != before[i].Content {
			t.Fatalf("history[%d] was mutated by a limit change", i)
		}
	}
	if summary != "a persisted rolling summary with EXACT FACTS: NEBULA-9634" {
		t.Fatal("the summary was modified")
	}
}

// The cache must not miss two rapid saves of the same length. "17" and "10"
// are the same number of bytes, and a coarse filesystem timestamp can place two
// saves in the same tick; a size-and-mtime key would then answer 17 forever.
// SaveConfig renames a temporary file over the target, so identity is what
// actually changes. This writes the two payloads byte-for-byte equal in length
// and pins the timestamp, so only identity differs.
func TestSameSizeRapidReplacementIsNotMissed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Same length by construction: only the two digits differ.
	const shape = `{"agents":{"defaults":{"telegram_recent_context_messages":%s}}}`
	writeAtomic := func(limit string) {
		tmp := filepath.Join(dir, ".tmp-"+limit)
		if err := os.WriteFile(tmp, []byte(fmt.Sprintf(shape, limit)), 0o600); err != nil {
			t.Fatal(err)
		}
		// The same temp-file-then-rename SaveConfig performs.
		if err := os.Rename(tmp, path); err != nil {
			t.Fatal(err)
		}
	}

	previous := telegramContextLimit
	telegramContextLimit = &liveTelegramContextLimit{path: path}
	t.Cleanup(func() { telegramContextLimit = previous })

	writeAtomic("17")
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	agent := telegramAgent(15)
	if got := recentContextLimit(agent, "telegram"); got != 17 {
		t.Fatalf("limit = %d, want 17", got)
	}

	writeAtomic("10")
	if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if after.Size() != before.Size() {
		t.Fatalf("fixture is wrong: sizes differ (%d vs %d)", before.Size(), after.Size())
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("could not pin the modification time; the test would prove nothing")
	}

	if got := recentContextLimit(agent, "telegram"); got != 10 {
		t.Fatalf("limit = %d after a same-size, same-mtime replacement, want 10", got)
	}
}

// os.SameFile is what makes the above work; pin it directly so a future
// simplification of the cache key cannot silently drop it.
func TestCacheKeyUsesFileIdentity(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "a")
	second := filepath.Join(dir, "b")
	if err := os.WriteFile(first, []byte("xx"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("yy"), 0o600); err != nil {
		t.Fatal(err)
	}
	infoA, err := os.Stat(first)
	if err != nil {
		t.Fatal(err)
	}
	infoB, err := os.Stat(second)
	if err != nil {
		t.Fatal(err)
	}
	// Identical size, identical timestamp, different files.
	if err := os.Chtimes(second, infoA.ModTime(), infoA.ModTime()); err != nil {
		t.Fatal(err)
	}
	infoB, err = os.Stat(second)
	if err != nil {
		t.Fatal(err)
	}

	if infoA.Size() != infoB.Size() || !infoA.ModTime().Equal(infoB.ModTime()) {
		t.Fatal("the fixture failed to make size and mtime identical")
	}
	if sameConfigFile(infoA, infoB) {
		t.Fatal("two different files compared equal; the cache key ignores identity")
	}
	if !sameConfigFile(infoA, infoA) {
		t.Fatal("a file did not compare equal to itself")
	}
}
