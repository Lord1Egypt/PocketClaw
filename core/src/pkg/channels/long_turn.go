package channels

import "time"

// LongTurnEditWindow is how old a turn's status message (the Thinking
// placeholder, or the tool-progress message it became) may be before the final
// answer is no longer edited into it.
//
// PC-DEF-084. Editing is right for a short turn: the answer replaces its own
// status line. After a long turn it is wrong on a chat app like Telegram, where
// an edit makes no sound and keeps the message's original position and time:
// the answer lands above anything the owner sent meanwhile, silently, and the
// turn reads as a hang. Past this window the answer is sent as a new message
// and the stale status message is deleted. That is one ordinary notification
// per turn and no periodic traffic.
const LongTurnEditWindow = 2 * time.Minute

// FinalEditWindowChannel is implemented by channels whose edits are silent and
// positional. The manager sends a final answer fresh, instead of editing a
// status message older than FinalEditWindow, only for these channels; any
// other channel keeps editing in place.
type FinalEditWindowChannel interface {
	FinalEditWindow() time.Duration
}

// statusMessageTooOldToEdit reports whether a status message sent at sentAt
// should be replaced by a fresh message rather than edited on ch.
func statusMessageTooOldToEdit(ch Channel, sentAt time.Time, now time.Time) bool {
	windowed, ok := ch.(FinalEditWindowChannel)
	if !ok || sentAt.IsZero() {
		return false
	}
	return now.Sub(sentAt) >= windowed.FinalEditWindow()
}
