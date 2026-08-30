package commands

import (
	"context"
	"strings"
)

func btwCommand() Definition {
	return Definition{
		Name:        "btw",
		Description: "Ask a side question without changing session history",
		Usage:       "/btw <question>",
		Handler: func(ctx context.Context, req Request, rt *Runtime) error {
			// Neutral wording on purpose: a contentless answer is not evidence
			// of a provider fault, and claiming one sends diagnosis the wrong
			// way. See the matching note on agent.defaultResponse.
			const emptyAnswerMsg = "The model returned no user-visible response."

			if rt == nil || rt.AskSideQuestion == nil {
				return req.Reply(unavailableMsg)
			}

			question := sideQuestionText(req.Text)
			if question == "" {
				return req.Reply("Usage: /btw <question>")
			}

			answer, err := rt.AskSideQuestion(ctx, question)
			if err != nil {
				return req.Reply(err.Error())
			}
			if strings.TrimSpace(answer) == "" {
				return req.Reply(emptyAnswerMsg)
			}

			return req.Reply(answer)
		},
	}
}

func sideQuestionText(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	parts := strings.Fields(input)
	if len(parts) < 2 {
		return ""
	}
	if !strings.HasPrefix(input, parts[0]) {
		return ""
	}
	return strings.TrimSpace(input[len(parts[0]):])
}
