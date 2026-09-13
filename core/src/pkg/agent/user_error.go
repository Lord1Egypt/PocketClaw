package agent

import (
	"errors"
	"fmt"
	"strings"
)

// UserFacingError is a failure already worded for the person who caused it.
//
// It exists so that a turn which cannot start says something the user can act
// on. Without it the only vocabulary available for a precondition failure was
// `fmt.Errorf`, whose text reaches the chat window through
// formatProcessingError's last branch as "Error processing message: <internal
// error>" -- or, where nothing raised an error at all, as silence.
//
// The message is primary. The code is a stable handle for support and for the
// dashboard to localise against: Core has no translation layer, so the English
// message is what Telegram shows, while the console can map the code to the
// user's language. A code is therefore never renamed once shipped.
//
// Anything placed in Message must already be safe. Nothing here redacts: the
// constructors below take fixed strings, and the one path that quotes a
// provider goes through providerErrorDetail's sanitisation instead.
type UserFacingError struct {
	// Code is a stable identifier, PC-E-<CATEGORY>-<NNN>.
	Code string
	// Message is the safe, actionable sentence shown to the user.
	Message string
	// cause is kept for the developer log and is never rendered to the user.
	cause error
}

func (e *UserFacingError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *UserFacingError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// UserMessage renders the error for a chat window: the sentence, then the code
// in parentheses so a user can quote it without the code displacing the advice.
func (e *UserFacingError) UserMessage() string {
	if e == nil {
		return ""
	}
	message := strings.TrimSpace(e.Message)
	if message == "" {
		return e.Code
	}
	if e.Code == "" {
		return message
	}
	return fmt.Sprintf("%s (%s)", message, e.Code)
}

// AsUserFacingError reports whether err carries a message already written for
// the user, and returns it.
func AsUserFacingError(err error) (*UserFacingError, bool) {
	var target *UserFacingError
	if errors.As(err, &target) && target != nil {
		return target, true
	}
	return nil, false
}

// Stable user-facing error codes.
//
// Grouped by what the user has to change, because that is the only distinction
// that changes what they do next. Never renumber or reuse one of these.
const (
	// Configuration: something is missing or stale in PocketClaw's own setup.
	CodeNoModelConfigured = "PC-E-AI-001"
	CodeNoModelSelected   = "PC-E-AI-002"
	CodeSelectedModelGone = "PC-E-AI-003"
	CodeNoModelEnabled    = "PC-E-AI-004"
)

// newNoModelConfigured is the first-run case: Telegram is connected and working,
// and no AI model has been added yet.
//
// The wording separates the two facts on purpose. A silent bot reads as a broken
// Telegram integration, and the owner then goes and re-pairs a bot that was
// never at fault.
func newNoModelConfigured() *UserFacingError {
	return &UserFacingError{
		Code: CodeNoModelConfigured,
		Message: "PocketClaw is connected, but no AI model is configured yet. " +
			"Open PocketClaw, add an AI provider and model, then send this again.",
	}
}

func newNoModelEnabled() *UserFacingError {
	return &UserFacingError{
		Code: CodeNoModelEnabled,
		Message: "PocketClaw is connected, but every configured AI model is disabled. " +
			"Open PocketClaw, enable a model, then send this again.",
	}
}

func newNoModelSelected() *UserFacingError {
	return &UserFacingError{
		Code: CodeNoModelSelected,
		Message: "PocketClaw is connected, but no default AI model is selected. " +
			"Open PocketClaw, choose a default model, then send this again.",
	}
}

func newSelectedModelGone(name string) *UserFacingError {
	return &UserFacingError{
		Code: CodeSelectedModelGone,
		Message: fmt.Sprintf(
			"The selected AI model %q is no longer configured -- its provider or model "+
				"entry was removed. Open PocketClaw and choose a model that still exists.",
			name),
	}
}

// ErrAINotConfigured is the fallback for a blocked provider that has no more
// specific explanation. It is a value rather than a constructor because callers
// reach for it when they have nothing to describe.
var ErrAINotConfigured = newNoModelConfigured()

// NewNoModelSelectedProblem is the exported constructor for the
// "nothing selected" case, used by the gateway when it starts in limited mode
// with a configuration that is otherwise complete.
func NewNoModelSelectedProblem() *UserFacingError { return newNoModelSelected() }
