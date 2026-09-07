package agent

import (
	"strings"
	"testing"
)

const modelCommandReply = "🤖 Model selection is managed from PocketClaw Settings."

// defaultHarnessAgent returns the single agent the cancellation harness builds,
// so a test can read the runtime model state /model must not touch.
func defaultHarnessAgent(t *testing.T, h *cancellationHarness) *AgentInstance {
	t.Helper()
	registry := h.loop.GetRegistry()
	ids := registry.ListAgentIDs()
	if len(ids) == 0 {
		t.Fatal("harness registered no agents")
	}
	agent, ok := registry.GetAgent(ids[0])
	if !ok {
		t.Fatalf("agent %q is registered but not retrievable", ids[0])
	}
	return agent
}

// anyHistory reports the total history recorded across every session, so a test
// need not know how the harness keys them.
func anyHistory(t *testing.T, agent *AgentInstance) int {
	t.Helper()
	if agent.Sessions == nil {
		t.Fatal("agent has no session store; a history assertion would be vacuous")
	}
	total := 0
	for _, key := range agent.Sessions.ListSessions() {
		total += len(agent.Sessions.GetHistory(key))
	}
	return total
}

func deliveredText(t *testing.T, h *cancellationHarness, count int) string {
	t.Helper()
	edits, sends := waitForDeliveryCount(t, h.channel, count)
	var all []string
	for _, edit := range edits {
		all = append(all, edit.content)
	}
	all = append(all, sends...)
	return strings.Join(all, "\n")
}

// 2 and 3. /model answers from a constant: it never reaches the provider, never
// moves the agent's runtime model, and writes nothing to the conversation.
//
// The second half of the test is what makes the first half mean anything. An
// ordinary message immediately afterwards must reach the provider and must be
// recorded, so "no calls, no history" is a property of /model rather than of a
// harness that never records anything.
func TestModelCommandNeitherCallsTheModelNorRecordsATurn(t *testing.T) {
	h := newCancellationHarness(t, "an ordinary answer")
	h.provider.releaseAll()

	agent := defaultHarnessAgent(t, h)
	modelBefore, providerBefore := agent.Model, agent.Provider

	h.send(t, "M", "/model")
	if reply := deliveredText(t, h, 1); !strings.Contains(reply, modelCommandReply) {
		t.Fatalf("/model delivered %q, want the fixed Dashboard message", reply)
	}

	if calls := h.provider.calls.Load(); calls != 0 {
		t.Fatalf("/model reached the provider %d time(s); it must answer from a constant", calls)
	}
	if agent.Model != modelBefore {
		t.Fatalf("/model changed the runtime model from %q to %q", modelBefore, agent.Model)
	}
	if agent.Provider != providerBefore {
		t.Fatal("/model replaced the runtime provider")
	}
	if recorded := anyHistory(t, agent); recorded != 0 {
		t.Fatalf("/model recorded %d history message(s); a command is not a conversational turn", recorded)
	}

	// Control: an ordinary message does both of the things /model did not, so
	// the assertions above cannot pass by the harness recording nothing.
	h.send(t, "O", "an ordinary question")
	if reply := deliveredText(t, h, 2); !strings.Contains(reply, "an ordinary answer") {
		t.Fatalf("the control message was not answered: %q", reply)
	}
	if calls := h.provider.calls.Load(); calls == 0 {
		t.Fatal("the control message never reached the provider; the no-call assertion proves nothing")
	}
	if recorded := anyHistory(t, agent); recorded == 0 {
		t.Fatal("the control message recorded no history; the no-history assertion proves nothing")
	}
}

// /model must not become a model setter through its arguments either.
func TestModelCommandIgnoresArgumentsAndStillChangesNothing(t *testing.T) {
	h := newCancellationHarness(t, "unused")
	h.provider.releaseAll()

	agent := defaultHarnessAgent(t, h)
	modelBefore := agent.Model

	h.send(t, "M", "/model some-other-model")
	if reply := deliveredText(t, h, 1); !strings.Contains(reply, modelCommandReply) {
		t.Fatalf("/model with an argument delivered %q, want the fixed message", reply)
	}
	if agent.Model != modelBefore {
		t.Fatalf("/model with an argument changed the model to %q", agent.Model)
	}
	if calls := h.provider.calls.Load(); calls != 0 {
		t.Fatalf("/model with an argument reached the provider %d time(s)", calls)
	}
}
