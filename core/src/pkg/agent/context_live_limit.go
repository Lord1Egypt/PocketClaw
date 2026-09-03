package agent

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

// liveTelegramContextLimit reads the Telegram context-memory limit from the
// configuration file at the moment a projection needs it.
//
// The limit cannot be answered from AgentInstance alone. Settings saves it in
// the web/launcher process, while the agent runs in the gateway child process,
// so an in-memory update on one side is invisible to the other. The agent's own
// config is a startup snapshot and is never reassigned at runtime, which is why
// a saved 17 kept projecting 15 until the service was restarted.
//
// The file is therefore the single source of truth, and this reads it — but
// only when it has actually changed. Each resolve does one stat; the value is
// re-decoded only when size or modification time differs from the last read.
// Nothing polls, no timer runs, and an unchanged file costs a stat and a mutex.
type liveTelegramContextLimit struct {
	mu       sync.Mutex
	path     string
	modTime  time.Time
	size     int64
	value    int
	haveRead bool
}

// telegramContextLimitFile decodes only the field this needs.
//
// config.LoadConfig is deliberately not used: it runs migration, resolver setup
// and diagnostic logging, none of which belong on a per-turn path, and any of
// which could log on every turn if the file were malformed.
type telegramContextLimitFile struct {
	Agents struct {
		Defaults struct {
			TelegramRecentContextMessages int `json:"telegram_recent_context_messages"`
		} `json:"defaults"`
	} `json:"agents"`
}

// resolve returns the effective limit, or fallback when the file cannot be read.
//
// A missing or unreadable config is not an error worth failing a turn over: the
// agent keeps using the value it started with, which is the same number the file
// would have produced in every normal case.
func (l *liveTelegramContextLimit) resolve(fallback int) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.path == "" {
		l.path = config.ResolveConfigPath()
	}

	info, err := os.Stat(l.path)
	if err != nil {
		return fallback
	}
	if l.haveRead && info.ModTime().Equal(l.modTime) && info.Size() == l.size {
		return l.value
	}

	raw, err := os.ReadFile(l.path)
	if err != nil {
		return fallback
	}
	var parsed telegramContextLimitFile
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// A half-written file during a save is transient. Keep the previous
		// answer rather than snapping to the default for one turn.
		if l.haveRead {
			return l.value
		}
		return fallback
	}

	l.value = config.ResolveTelegramRecentContextMessages(
		parsed.Agents.Defaults.TelegramRecentContextMessages,
	)
	l.modTime = info.ModTime()
	l.size = info.Size()
	l.haveRead = true
	return l.value
}

// telegramContextLimit is process-wide because the file is. One agent loop runs
// per gateway process, and a shared cache means a single stat per projection
// however many sessions are active.
var telegramContextLimit = &liveTelegramContextLimit{}
