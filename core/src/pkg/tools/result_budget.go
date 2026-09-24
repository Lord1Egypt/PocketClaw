package tools

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ResultBudget is what one tool result looks like after it has been bounded
// for the model.
type ResultBudget struct {
	Content        string
	OriginalBytes  int
	DeliveredBytes int
	Truncated      bool
}

// resultHeadShare is how much of a truncated result's budget goes to its
// beginning. The head carries the exit status, headers and the start of a
// structure; the tail carries the end of a log, a pagination hint, or the
// error that stopped a command — both are worth keeping, the head a little more.
const resultHeadShare = 0.7

// resultLineSnapWindow is how far a cut may move to land on a line break
// rather than mid-line, as a fraction of that side's budget.
const resultLineSnapWindow = 8

// BoundResultForLLM limits content to maxBytes of valid UTF-8.
//
// A result that fits is returned unchanged. One that does not keeps its
// beginning and its end, and the removed middle is replaced by a notice that
// states both sizes, so the model can never mistake a partial result for a
// complete one. The notice is part of the budget: the returned content is never
// longer than maxBytes.
//
// Invalid UTF-8 — binary output, or a capture buffer that stopped inside a
// multi-byte sequence — is repaired before cutting, and every cut lands on a
// rune boundary, so the result is always valid UTF-8 and never splits a
// character.
func BoundResultForLLM(content string, maxBytes int) ResultBudget {
	original := len(content)
	if !utf8.ValidString(content) {
		content = strings.ToValidUTF8(content, "�")
	}
	if maxBytes <= 0 || len(content) <= maxBytes {
		return ResultBudget{
			Content:        content,
			OriginalBytes:  original,
			DeliveredBytes: len(content),
		}
	}

	// Every number in the notice is at most the original size, so a notice
	// rendered with the original size in every slot is the longest it can be.
	longestNotice := resultTruncationNotice(original, original, original, original)
	available := maxBytes - len(longestNotice)
	if available < 2 {
		// A budget too small to carry the notice: keep the notice alone rather
		// than silently dropping the fact of truncation.
		notice := resultTruncationNotice(original, 0, 0, 0)
		return ResultBudget{Content: notice, OriginalBytes: original, Truncated: true}
	}

	headBudget := int(float64(available) * resultHeadShare)
	tailBudget := available - headBudget
	head := content[:cutHeadAt(content, headBudget)]
	tail := content[cutTailAt(content, tailBudget):]
	delivered := len(head) + len(tail)
	notice := resultTruncationNotice(original, delivered, len(head), len(tail))

	return ResultBudget{
		Content:        head + notice + tail,
		OriginalBytes:  original,
		DeliveredBytes: delivered,
		Truncated:      true,
	}
}

// resultTruncationNotice is the marker placed where the middle was removed.
// Its first line is fixed text so it is easy to recognise, and it tells the
// model what to do instead of repeating the same command.
func resultTruncationNotice(original, delivered, head, tail int) string {
	return fmt.Sprintf(
		"\n\n[OUTPUT TRUNCATED]\n"+
			"original bytes: %d\n"+
			"delivered bytes: %d (first %d + last %d; the middle was removed)\n"+
			"This is not the complete output. Narrow the request instead of repeating it: "+
			"filter JSON with jq, select lines with rg, grep, head or tail, ask an API for "+
			"fewer fields or a smaller page, or save a large download into the workspace "+
			"(curl -o FILE) and inspect it in parts.\n\n",
		original, delivered, head, tail,
	)
}

// cutHeadAt returns an index <= limit that ends the head on a rune boundary,
// preferring the end of a line when one is close.
func cutHeadAt(content string, limit int) int {
	if limit >= len(content) {
		return len(content)
	}
	end := limit
	for end > 0 && !utf8.RuneStart(content[end]) {
		end--
	}
	if newline := strings.LastIndexByte(content[:end], '\n'); newline >= 0 &&
		end-newline <= limit/resultLineSnapWindow {
		return newline + 1
	}
	return end
}

// cutTailAt returns an index such that content[index:] is at most limit bytes,
// starts on a rune boundary and, when one is close, at the start of a line.
func cutTailAt(content string, limit int) int {
	if limit >= len(content) {
		return 0
	}
	start := len(content) - limit
	for start < len(content) && !utf8.RuneStart(content[start]) {
		start++
	}
	if newline := strings.IndexByte(content[start:], '\n'); newline >= 0 &&
		newline < limit/resultLineSnapWindow {
		return start + newline + 1
	}
	return start
}
