package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

const testBudget = 64 * 1024

func TestBoundResultForLLMLeavesSmallOutputAlone(t *testing.T) {
	in := "curl exited 0 after 12ms (completed)\n\nstdout:\nhello\n"
	got := BoundResultForLLM(in, testBudget)
	if got.Truncated || got.Content != in {
		t.Fatalf("small output changed: truncated=%v content=%q", got.Truncated, got.Content)
	}
	if got.OriginalBytes != len(in) || got.DeliveredBytes != len(in) {
		t.Fatalf("sizes = %d/%d, want %d/%d", got.OriginalBytes, got.DeliveredBytes, len(in), len(in))
	}
}

// The incident: curl returned 3,202,895 bytes, the runtime's 4 MiB capture cap
// did not trip, and the whole body went into the next provider request.
func TestBoundResultForLLMBoundsAMultiMegabyteStdout(t *testing.T) {
	header := "curl exited 0 after 2140ms (completed)\n\nstdout:\n"
	footer := "\nEND-OF-RESPONSE-MARKER\n"
	body := strings.Repeat("0123456789abcdef", 3_202_895/16)
	in := header + body + footer

	got := BoundResultForLLM(in, testBudget)

	if !got.Truncated {
		t.Fatal("a 3 MB result was not truncated")
	}
	if len(got.Content) > testBudget {
		t.Fatalf("delivered %d bytes, budget is %d", len(got.Content), testBudget)
	}
	if !strings.HasPrefix(got.Content, header) {
		t.Fatal("the beginning — and with it the exit status — was not preserved")
	}
	if !strings.HasSuffix(got.Content, footer) {
		t.Fatal("the end of the output was not preserved")
	}
	for _, want := range []string{
		"[OUTPUT TRUNCATED]",
		fmt.Sprintf("original bytes: %d", len(in)),
		fmt.Sprintf("delivered bytes: %d", got.DeliveredBytes),
	} {
		if !strings.Contains(got.Content, want) {
			t.Fatalf("truncation notice is missing %q", want)
		}
	}
	if got.OriginalBytes != len(in) {
		t.Fatalf("OriginalBytes = %d, want %d", got.OriginalBytes, len(in))
	}
	if got.DeliveredBytes <= 0 || got.DeliveredBytes >= len(in) {
		t.Fatalf("DeliveredBytes = %d is not a strict part of %d", got.DeliveredBytes, len(in))
	}
}

// Every cut must land on a rune boundary, whatever the budget and whatever the
// mix of one- to four-byte characters around it.
func TestBoundResultForLLMNeverSplitsAUTF8Character(t *testing.T) {
	pieces := []string{"a", "é", "漢", "😀", "\n", "ب"}
	var b strings.Builder
	for i := 0; b.Len() < 200_000; i++ {
		b.WriteString(pieces[i%len(pieces)])
	}
	in := b.String()

	for budget := 4096; budget < 4096+64; budget++ {
		got := BoundResultForLLM(in, budget)
		if !utf8.ValidString(got.Content) {
			t.Fatalf("budget %d produced invalid UTF-8", budget)
		}
		if len(got.Content) > budget {
			t.Fatalf("budget %d delivered %d bytes", budget, len(got.Content))
		}
	}
}

// A capture buffer that stopped inside a multi-byte sequence — the runtime's
// bounded buffer cuts at a byte count — must still yield valid UTF-8.
func TestBoundResultForLLMRepairsMalformedInput(t *testing.T) {
	cut := "漢"[:2]
	in := strings.Repeat("x", 100_000) + cut + strings.Repeat("y", 100_000) + cut

	got := BoundResultForLLM(in, testBudget)
	if !utf8.ValidString(got.Content) {
		t.Fatal("malformed input produced invalid UTF-8")
	}
	if got.OriginalBytes != len(in) {
		t.Fatalf("OriginalBytes = %d, want the raw size %d", got.OriginalBytes, len(in))
	}

	small := BoundResultForLLM("ok"+cut, testBudget)
	if !utf8.ValidString(small.Content) || small.Truncated {
		t.Fatalf("small malformed input: valid=%v truncated=%v",
			utf8.ValidString(small.Content), small.Truncated)
	}
}

// A synthetic `gh api repos/o/r/git/trees/HEAD?recursive=1`: the shape that
// commonly produces multi-megabyte JSON from one innocent-looking call.
func TestBoundResultForLLMKeepsBothEndsOfAGiantGitHubTree(t *testing.T) {
	type entry struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
		Type string `json:"type"`
		SHA  string `json:"sha"`
		Size int    `json:"size"`
		URL  string `json:"url"`
	}
	entries := make([]entry, 0, 20_000)
	for i := 0; i < 20_000; i++ {
		entries = append(entries, entry{
			Path: fmt.Sprintf("src/module_%05d/file_%05d.go", i/50, i),
			Mode: "100644",
			Type: "blob",
			SHA:  fmt.Sprintf("%040x", i),
			Size: 1000 + i,
			URL:  fmt.Sprintf("https://api.github.com/repos/o/r/git/blobs/%040x", i),
		})
	}
	// A struct, not a map: GitHub puts "truncated" last, and json.Marshal
	// would sort a map's keys.
	tree, err := json.Marshal(struct {
		SHA       string  `json:"sha"`
		URL       string  `json:"url"`
		Tree      []entry `json:"tree"`
		Truncated bool    `json:"truncated"`
	}{
		SHA:  strings.Repeat("a", 40),
		URL:  "https://api.github.com/repos/o/r/git/trees/HEAD",
		Tree: entries,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) < 3_000_000 {
		t.Fatalf("synthetic tree is only %d bytes; the test needs a multi-megabyte result", len(tree))
	}
	in := "gh exited 0 after 3100ms (completed)\n\nstdout:\n" + string(tree) + "\n"

	got := BoundResultForLLM(in, testBudget)
	if len(got.Content) > testBudget {
		t.Fatalf("delivered %d bytes, budget is %d", len(got.Content), testBudget)
	}
	if !strings.HasPrefix(got.Content, "gh exited 0") {
		t.Fatal("exit status line lost")
	}
	if !strings.Contains(got.Content, `"sha":"`+strings.Repeat("a", 40)) {
		t.Fatal("start of the JSON document lost")
	}
	if !strings.Contains(got.Content, `"truncated":false}`) {
		t.Fatal("end of the JSON document lost")
	}
	if !strings.Contains(got.Content, "jq") {
		t.Fatal("the notice does not tell the model how to narrow a JSON result")
	}
}

func TestBoundResultForLLMStillReportsTruncationUnderATinyBudget(t *testing.T) {
	got := BoundResultForLLM(strings.Repeat("z", 10_000), 64)
	if !got.Truncated || !strings.Contains(got.Content, "[OUTPUT TRUNCATED]") {
		t.Fatalf("tiny budget dropped the truncation notice: %q", got.Content)
	}
}
