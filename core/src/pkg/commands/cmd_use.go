package commands

func useCommand() Definition {
	return Definition{
		Name:        "use",
		Description: "Use a specific skill for your next request",
		Usage:       "/use <skill> [message]",
	}
}
