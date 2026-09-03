package agent

import (
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/providers"
)

// summarize simulates a summarizer that compressed the segment semantically and
// dropped every literal — exactly the failure observed on device.
func semanticOnlySummary() string {
	return "The user supplied a temporary test code and asked about a branch and a build."
}

func factValues(t *testing.T, batch []providers.Message) []string {
	t.Helper()
	return sortedFactValues(extractExactFacts(batch))
}

func containsValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// THE PHYSICAL FAILURE. A session-only identifier stated by the user must come
// back verbatim even when the summarizer wrote only a description of it.
func TestExactIdentifierSurvivesASemanticSummary(t *testing.T) {
	batch := []providers.Message{
		userMsg("For this session only: the test code is ORBIT-4826. " +
			"Do not save it to memory and do not use a memory tool."),
		assistantMsg("Understood, session context only."),
	}

	summary := ensureExactFacts(semanticOnlySummary(), extractExactFacts(batch))

	if !strings.Contains(summary, "ORBIT-4826") {
		t.Fatalf("ORBIT-4826 did not survive summarization; summary = %q", summary)
	}
	if !strings.Contains(summary, exactFactsHeading) {
		t.Errorf("restored facts were not marked as exact; summary = %q", summary)
	}
}

func TestTechnicalLiteralsSurvive(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    string
	}{
		{"branch", "Work on branch feature/telegram-bounded-context please.",
			"feature/telegram-bounded-context"},
		{"commit", "The commit is 4e87077a6c67e46fa9d901dfc8757ab6db233933.",
			"4e87077a6c67e46fa9d901dfc8757ab6db233933"},
		{"version code", "Build versionCode 29 next.", "29"},
		{"port", "The gateway listens on port 18790.", "18790"},
		{"path", "Read /home/lordegypt/PocketClaw-App/core/src for the source.",
			"/home/lordegypt/PocketClaw-App/core/src"},
		{"ticket code", "Track it under ORBIT-4826.", "ORBIT-4826"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			batch := []providers.Message{userMsg(tc.message)}
			values := factValues(t, batch)
			if !containsValue(values, tc.want) {
				t.Fatalf("%q was not extracted from %q; got %v", tc.want, tc.message, values)
			}

			summary := ensureExactFacts(semanticOnlySummary(), extractExactFacts(batch))
			if !strings.Contains(summary, tc.want) {
				t.Fatalf("%q did not reach the summary; summary = %q", tc.want, summary)
			}
		})
	}
}

// A fact must survive being re-summarized, or it disappears one cycle after it
// was stated rather than immediately.
func TestFactsSurviveMultipleSummaryGenerations(t *testing.T) {
	first := []providers.Message{
		userMsg("Session test code is ORBIT-4826 and the branch is feature/telegram-bounded-context."),
	}
	summary1 := ensureExactFacts(semanticOnlySummary(), extractExactFacts(first))
	if !strings.Contains(summary1, "ORBIT-4826") {
		t.Fatal("generation 1 lost ORBIT-4826")
	}

	// Generation 2: a new batch about something else, summarized semantically
	// again. The earlier facts must be carried, not re-derived from messages
	// that no longer exist.
	second := []providers.Message{userMsg("Now check the logs and tell me what you see.")}
	carried := carryForwardExactFacts(summary1, extractExactFacts(second))
	summary2 := ensureExactFacts("The user asked about logs.", carried)

	if !strings.Contains(summary2, "ORBIT-4826") {
		t.Fatalf("generation 2 lost ORBIT-4826; summary = %q", summary2)
	}
	if !strings.Contains(summary2, "feature/telegram-bounded-context") {
		t.Fatalf("generation 2 lost the branch; summary = %q", summary2)
	}

	// Generation 3, to prove it is not a one-off.
	third := []providers.Message{userMsg("Anything else worth noting?")}
	summary3 := ensureExactFacts(
		"The user asked a follow-up.",
		carryForwardExactFacts(summary2, extractExactFacts(third)),
	)
	if !strings.Contains(summary3, "ORBIT-4826") {
		t.Fatalf("generation 3 lost ORBIT-4826; summary = %q", summary3)
	}
}

// A restated value replaces the old one rather than accumulating both.
func TestNewerValueSupersedesTheOlderOne(t *testing.T) {
	first := []providers.Message{userMsg("Build versionCode 29 for the candidate.")}
	summary1 := ensureExactFacts("Summary.", extractExactFacts(first))
	if !strings.Contains(summary1, "versionCode: 29") {
		t.Fatalf("versionCode 29 was not recorded; summary = %q", summary1)
	}

	newer := []providers.Message{userMsg("Scratch that, use versionCode 30 instead.")}
	summary2 := ensureExactFacts(
		"Summary.",
		carryForwardExactFacts(summary1, extractExactFacts(newer)),
	)

	if !strings.Contains(summary2, "versionCode: 30") {
		t.Fatalf("the newer versionCode was not kept; summary = %q", summary2)
	}
	if strings.Contains(summary2, "versionCode: 29") {
		t.Fatalf("the stale versionCode was kept alongside the new one; summary = %q", summary2)
	}
}

// Filler must stay compressed. The block is for identity, not transcription.
func TestNarrativeChatterIsNotQuotedVerbatim(t *testing.T) {
	batch := []providers.Message{
		userMsg("thanks, that is great"),
		userMsg("ok"),
		userMsg("sure, go ahead when you are ready"),
		userMsg("the answer is yes"),
		assistantMsg("Happy to help."),
	}

	facts := extractExactFacts(batch)
	if len(facts) != 0 {
		t.Fatalf("chatter produced %d facts: %v", len(facts), sortedFactValues(facts))
	}

	summary := "The user thanked the assistant and confirmed."
	if got := ensureExactFacts(summary, facts); got != summary {
		t.Fatalf("chatter changed the summary: %q", got)
	}
}

// The summary must not become a credential vault. This defers to the runtime's
// existing secret policy rather than a second list, so it cannot drift from it.
func TestCredentialsNeverEnterTheSummary(t *testing.T) {
	secrets := []string{
		"the github token is ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345",
		"api key is sk-ABCDEFGHIJKLMNOPQRSTUVWXYZ012345",
		"the telegram token is 123456789:AAEEfhbJHGSDFhjksdhfjkshdfjkhsdkfjhsd",
		"Authorization: Bearer abcdefghijklmnopqrstuvwxyz012345",
		"set password = hunter2correcthorsebattery",
	}

	for _, message := range secrets {
		batch := []providers.Message{userMsg(message)}
		facts := extractExactFacts(batch)
		summary := ensureExactFacts("A credential was discussed.", facts)

		for _, token := range strings.Fields(message) {
			if len(token) < 16 {
				continue
			}
			if strings.Contains(summary, strings.Trim(token, ".,;:")) {
				t.Errorf("credential material %q reached the summary from %q", token, message)
			}
		}
	}
}

// A well-behaved summarizer that already quoted the value must not have it
// appended a second time.
func TestFactsAlreadyPresentAreNotDuplicated(t *testing.T) {
	batch := []providers.Message{userMsg("The test code is ORBIT-4826.")}
	summary := "Temporary test code: ORBIT-4826. The user is running a context test."

	got := ensureExactFacts(summary, extractExactFacts(batch))
	if got != summary {
		t.Fatalf("a summary that already held the fact was rewritten: %q", got)
	}
	if strings.Count(got, "ORBIT-4826") != 1 {
		t.Fatalf("ORBIT-4826 appears %d times, want 1", strings.Count(got, "ORBIT-4826"))
	}
}

// Only what the user stated is quoted. Mining assistant prose would fill the
// block with identifiers the agent produced itself.
func TestOnlyUserStatedFactsAreExtracted(t *testing.T) {
	batch := []providers.Message{
		assistantMsg("I built versionCode 99 on branch feature/invented-by-me."),
	}
	if facts := extractExactFacts(batch); len(facts) != 0 {
		t.Fatalf("assistant output produced %d facts: %v", len(facts), sortedFactValues(facts))
	}
}

// The block is bounded; a long session cannot grow it without limit.
func TestExactFactsBlockIsBounded(t *testing.T) {
	var batch []providers.Message
	for i := 0; i < maxExactFacts*4; i++ {
		batch = append(batch, userMsg(
			strings.Repeat("x", 0)+"code"+strings.Repeat("A", i%3+3)+" is VALUE-"+
				strings.Repeat("1", i%5+2)+string(rune('a'+i%26)),
		))
	}
	if got := len(extractExactFacts(batch)); got > maxExactFacts {
		t.Fatalf("extracted %d facts, want at most %d", got, maxExactFacts)
	}

	oversized := []providers.Message{
		userMsg("blob is " + strings.Repeat("z", maxExactFactChars+50)),
	}
	for _, fact := range extractExactFacts(oversized) {
		if len(fact.Value) > maxExactFactChars {
			t.Fatalf("kept a %d-char value, want at most %d", len(fact.Value), maxExactFactChars)
		}
	}
}

func TestParseExactFactsRoundTrip(t *testing.T) {
	original := []exactFact{
		{Label: "test code", Value: "ORBIT-4826"},
		{Label: "versionCode", Value: "29"},
		{Value: "feature/telegram-bounded-context"},
	}
	summary := ensureExactFacts("Narrative.", original)

	parsed := parseExactFacts(summary)
	if len(parsed) != len(original) {
		t.Fatalf("parsed %d facts, want %d: %q", len(parsed), len(original), summary)
	}
	for i, fact := range original {
		if parsed[i].Value != fact.Value || parsed[i].Label != fact.Label {
			t.Errorf("fact %d round-tripped as %+v, want %+v", i, parsed[i], fact)
		}
	}
}
