package agent

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sipeed/picoclaw/pkg/pcruntime"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// exactFactsHeading introduces the verbatim block appended to a rolling summary.
//
// It is a heading rather than prose so a later summarization pass can recognise
// the block it is being asked to carry forward, and so a reader can tell what is
// quoted from the conversation from what the summarizer wrote about it.
const exactFactsHeading = "EXACT FACTS:"

const (
	// maxExactFacts bounds the block. A summary that accumulated every literal
	// ever mentioned would grow without limit, which is what the rolling summary
	// exists to prevent.
	maxExactFacts = 24
	// maxExactFactChars bounds one entry, so a pasted blob cannot enter the
	// summary through a fact.
	maxExactFactChars = 120
)

var (
	// labelledFactPattern captures "label: value" and "label = value" and
	// "label is value" — the shape a user uses when stating a fact that later
	// turns depend on. The label is what makes supersession possible: a second
	// "versionCode 30" replaces "versionCode 29" instead of accumulating.
	labelledFactPattern = regexp.MustCompile(
		`(?i)\b([A-Za-z][A-Za-z0-9 _./-]{1,40}?)\s*(?:=|:|\bis\b|\bwas\b)\s+` +
			`([A-Za-z0-9][A-Za-z0-9._/+@-]{1,80})\b`,
	)

	// bareIdentifierPatterns are literals worth quoting even with no label,
	// because their shape alone marks them as technical and exact.
	bareIdentifierPatterns = []*regexp.Regexp{
		// Ticket- or code-style identifiers: ORBIT-4826, JIRA-1234.
		regexp.MustCompile(`\b[A-Z][A-Z0-9]{2,15}-[A-Z0-9]{2,15}\b`),
		// Branch names under the conventional prefixes this project uses.
		regexp.MustCompile(
			`\b(?:feature|fix|cleanup|audit|docs|chore|release|hotfix)/[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)*`,
		),
		// Reverse-DNS package identifiers.
		regexp.MustCompile(`\b[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]{1,})\{2,\}`),
		// Absolute paths.
		regexp.MustCompile(`(?:^|\s)(/[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)+)`),
	}

	// juxtaposedFactPattern captures "versionCode 29" and "port 18790" — a
	// technical noun followed by its value with no separator, which is how these
	// are normally written.
	juxtaposedFactPattern = regexp.MustCompile(
		`(?i)\b(version[A-Za-z]*|port|branch|tag|commit|package|revision)\s+` +
			`([A-Za-z0-9][A-Za-z0-9._/+@-]{0,80})\b`,
	)

	// valueShapePattern is what separates a value from the next ordinary word.
	// Without it "Build versionCode 29" reads as the label "Build" holding the
	// value "versionCode", and the number is lost.
	valueShapePattern = regexp.MustCompile(`[0-9]|[./_@+]|-`)

	// secretLabelPattern mirrors the vocabulary the runtime's redaction uses for
	// secret-bearing keys. RedactText recognises credential *values* by shape,
	// which misses an ordinary-looking password stated in prose; this catches it
	// by what the user called it. Together they are strictly more careful than
	// either alone, and neither relaxes the other.
	// The bare words "session" and "auth" are deliberately absent: a user saying
	// "for this session only" is describing scope, not naming a credential, and
	// treating that as a secret would discard the very facts this exists to
	// keep. Only the compound forms that actually name credential material are
	// listed.
	secretLabelPattern = regexp.MustCompile(
		`(?i)(token|secret|password|passwd|passphrase|credential|` +
			`api[_ -]?key|private[_ -]?key|access[_ -]?key|cookie|` +
			`session[_ -](?:token|key|cookie|secret)|auth[_ -]?(?:token|key|header))`,
	)

	// factLabelPrefixPattern trims the conjunctions and articles that a label
	// picks up when it is lifted out of a sentence: "and the port" is "port".
	factLabelPrefixPattern = regexp.MustCompile(
		`(?i)^(?:and|but|so|then|also|the|a|an|our|your|my|its|this|that|new|current)\s+`,
	)

	// hexIdentifierPattern matches commit SHAs and content hashes. It is applied
	// separately because a bare hex run needs a length check to avoid matching
	// ordinary words.
	hexIdentifierPattern = regexp.MustCompile(`\b[0-9a-f]{7,64}\b`)

	// factLabelNoisePattern drops labels that describe the conversation rather
	// than name a fact, so "the answer is yes" does not become a fact.
	factLabelNoisePattern = regexp.MustCompile(
		`(?i)^(?:it|this|that|there|here|the answer|the result|the problem|the question|what|who|why|how|one|a|an|the)$`,
	)

	// factValueNoisePattern drops values that carry no identity.
	factValueNoisePattern = regexp.MustCompile(
		`(?i)^(?:yes|no|ok|okay|true|false|done|fine|good|sure|maybe|null|nil|none|it|this|that|now|then|here|there)$`,
	)
)

// exactFact is one verbatim literal worth carrying forward, with the label that
// lets a later statement supersede it.
type exactFact struct {
	Label string // normalized; empty for a bare identifier
	Value string // verbatim, exactly as the user wrote it
}

func (f exactFact) render() string {
	if f.Label == "" {
		return f.Value
	}
	return fmt.Sprintf("%s: %s", f.Label, f.Value)
}

// key is what supersession compares. Two statements of the same label are the
// same fact with a new value; two bare identifiers are the same only when
// identical.
func (f exactFact) key() string {
	if f.Label == "" {
		return "=" + f.Value
	}
	return strings.ToLower(f.Label)
}

// looksLikeCredential defers to the runtime's existing secret policy rather than
// keeping a second list of patterns that could drift from it.
//
// RedactText rewrites anything it recognises as credential material, so a
// candidate that comes back changed is one the project already treats as a
// secret and must never be copied into a summary. The rolling summary is not a
// credential vault, and this is the check that keeps it from becoming one.
func looksLikeCredential(text string) bool {
	return pcruntime.RedactText(text) != text
}

// statesASecret reports whether a message is assigning something the user
// called a credential. "set password = hunter2" carries no credential-shaped
// token for RedactText to recognise, but it is plainly a secret.
func statesASecret(content string) bool {
	for _, match := range labelledFactPattern.FindAllStringSubmatch(content, -1) {
		if secretLabelPattern.MatchString(match[1]) {
			return true
		}
	}
	return false
}

// extractExactFacts pulls verbatim literals worth preserving out of a batch
// about to be summarized.
//
// Only user messages are read. A user stating "the test code is ORBIT-4826" is
// supplying a fact that later turns depend on; assistant prose repeating it is
// not a new fact, and mining assistant output would fill the block with
// identifiers the agent itself produced.
//
// Later statements win, so a value restated in the same batch supersedes the
// earlier one.
func extractExactFacts(batch []providers.Message) []exactFact {
	ordered := make([]string, 0, maxExactFacts)
	byKey := make(map[string]exactFact, maxExactFacts)

	add := func(fact exactFact) {
		if !acceptableFact(fact) {
			return
		}
		key := fact.key()
		if _, seen := byKey[key]; !seen {
			ordered = append(ordered, key)
		}
		byKey[key] = fact // a later statement supersedes an earlier one
	}

	for _, msg := range batch {
		if msg.Role != "user" || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		if looksLikeCredential(msg.Content) || statesASecret(msg.Content) {
			// The message carries credential material. Nothing from it is
			// quoted: a fact extracted from the same sentence could easily be
			// the secret's neighbour, or the secret itself in another shape.
			continue
		}
		for _, match := range labelledFactPattern.FindAllStringSubmatch(msg.Content, -1) {
			add(exactFact{Label: normalizeFactLabel(match[1]), Value: match[2]})
		}
		for _, match := range juxtaposedFactPattern.FindAllStringSubmatch(msg.Content, -1) {
			add(exactFact{Label: normalizeFactLabel(match[1]), Value: match[2]})
		}
		for _, pattern := range bareIdentifierPatterns {
			for _, match := range pattern.FindAllStringSubmatch(msg.Content, -1) {
				value := match[len(match)-1]
				add(exactFact{Value: strings.TrimSpace(value)})
			}
		}
		for _, match := range hexIdentifierPattern.FindAllString(msg.Content, -1) {
			add(exactFact{Value: match})
		}
	}

	// A labelled fact already carries its value, so the same literal picked up
	// again as a bare identifier would list it twice.
	labelled := make(map[string]struct{}, len(byKey))
	for _, fact := range byKey {
		if fact.Label != "" {
			labelled[fact.Value] = struct{}{}
		}
	}

	facts := make([]exactFact, 0, len(ordered))
	for _, key := range ordered {
		fact := byKey[key]
		if fact.Label == "" {
			if _, covered := labelled[fact.Value]; covered {
				continue
			}
		}
		facts = append(facts, fact)
		if len(facts) == maxExactFacts {
			break
		}
	}
	return facts
}

func normalizeFactLabel(raw string) string {
	label := strings.TrimSpace(raw)
	for {
		trimmed := factLabelPrefixPattern.ReplaceAllString(label, "")
		if trimmed == label {
			break
		}
		label = trimmed
	}
	// A label is the last few words before the separator; anything longer is a
	// sentence, not a name.
	if fields := strings.Fields(label); len(fields) > 3 {
		label = strings.Join(fields[len(fields)-3:], " ")
	}
	return strings.TrimSpace(label)
}

func acceptableFact(fact exactFact) bool {
	value := strings.TrimSpace(fact.Value)
	if value == "" || len(value) > maxExactFactChars {
		return false
	}
	if factValueNoisePattern.MatchString(value) {
		return false
	}
	if fact.Label != "" && factLabelNoisePattern.MatchString(strings.TrimSpace(fact.Label)) {
		return false
	}
	if looksLikeCredential(value) || (fact.Label != "" && looksLikeCredential(fact.Label)) {
		return false
	}
	// A value with no digit and no separator is an ordinary word, not an
	// identifier — including the next word after a technical noun.
	if !valueShapePattern.MatchString(value) {
		return false
	}
	// Whatever the user called a token, secret, password or key is not quoted,
	// however innocuous its shape.
	if fact.Label != "" && secretLabelPattern.MatchString(fact.Label) {
		return false
	}
	return true
}

// ensureExactFacts appends the facts a summary failed to keep.
//
// The summarizer is asked to preserve these literals and usually does; this is
// the guarantee rather than the mechanism. A fact already present verbatim is
// left alone, so a well-behaved summary gains nothing and the block stays small.
func ensureExactFacts(summary string, facts []exactFact) string {
	if len(facts) == 0 {
		return summary
	}

	var missing []string
	for _, fact := range facts {
		if strings.Contains(summary, fact.Value) {
			continue
		}
		missing = append(missing, fact.render())
	}
	if len(missing) == 0 {
		return summary
	}

	var sb strings.Builder
	sb.WriteString(strings.TrimRight(summary, "\n"))
	if sb.Len() > 0 {
		sb.WriteString("\n\n")
	}
	sb.WriteString(exactFactsHeading)
	for _, entry := range missing {
		sb.WriteString("\n- ")
		sb.WriteString(entry)
	}
	return sb.String()
}

// carryForwardExactFacts keeps the previous summary's verbatim block alive
// across a re-summarization, unless the new batch supersedes an entry.
//
// Without this a fact would survive exactly one summarization: the next pass
// reads the summary as prose and compresses it away again, which is how an
// identifier disappears two cycles after it was stated.
func carryForwardExactFacts(previousSummary string, fresh []exactFact) []exactFact {
	carried := parseExactFacts(previousSummary)
	if len(carried) == 0 {
		return fresh
	}

	superseded := make(map[string]struct{}, len(fresh))
	for _, fact := range fresh {
		superseded[fact.key()] = struct{}{}
	}

	merged := make([]exactFact, 0, len(carried)+len(fresh))
	for _, fact := range carried {
		if _, replaced := superseded[fact.key()]; replaced {
			continue
		}
		merged = append(merged, fact)
	}
	merged = append(merged, fresh...)
	if len(merged) > maxExactFacts {
		// Keep the newest: the tail is what the recent conversation established.
		merged = merged[len(merged)-maxExactFacts:]
	}
	return merged
}

// parseExactFacts reads back a block written by ensureExactFacts.
func parseExactFacts(summary string) []exactFact {
	index := strings.Index(summary, exactFactsHeading)
	if index < 0 {
		return nil
	}
	var facts []exactFact
	for _, line := range strings.Split(summary[index+len(exactFactsHeading):], "\n") {
		entry := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if entry == "" {
			continue
		}
		label, value, found := strings.Cut(entry, ": ")
		if !found {
			facts = append(facts, exactFact{Value: entry})
			continue
		}
		facts = append(facts, exactFact{
			Label: strings.TrimSpace(label),
			Value: strings.TrimSpace(value),
		})
	}
	return facts
}

// sortedFactValues is a test and diagnostic helper.
func sortedFactValues(facts []exactFact) []string {
	values := make([]string, 0, len(facts))
	for _, fact := range facts {
		values = append(values, fact.Value)
	}
	sort.Strings(values)
	return values
}
