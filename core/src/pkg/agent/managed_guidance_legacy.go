package agent

// Historical bytes. Do not edit.
//
// This is the "## Managed Runtime" section exactly as PocketClaw seeded it into
// AGENT.md, from the first Managed Runtime release through 0.2.0+58. It is
// pinned here as a literal rather than derived from the live guidance for one
// reason: the live guidance changes, and these bytes cannot. They identify what
// is already on users' devices.
//
// The digest of this text is what decides whether an install's inline copy is
// PocketClaw's own — and therefore droppable from the prompt — or the user's,
// and therefore untouchable. Regenerating it to match reworded text would make
// it match nothing, silently restoring the duplicate for every existing
// install, or match something new, which is worse.
//
// COMPATIBILITY: contains the word PocketClaw and the product tool names.
// RECHECK AFTER FULL NAMESPACE MIGRATION — and even then, do not rewrite it:
// the bytes on a user's disk do not change when the namespace does.
const legacyManagedRuntimeSectionV1 = "" +
	"## Managed Runtime\n" +
	"\n" +
	"PocketClaw has a Managed Runtime: a catalog of verified command-line tools you\n" +
	"reach through the `runtime` tool. Use `action=list` to see what this device\n" +
	"actually provides before concluding that a command is missing. A tool that is\n" +
	"not on your PATH may still be available through the runtime.\n" +
	"\n" +
	"Four things to hold on to:\n" +
	"\n" +
	"- Do not assume \"command not found\". Ask the runtime first.\n" +
	"- The runtime cannot install software, and neither can you. On Android an\n" +
	"  executable runs only from the app package or the system image, both fixed when\n" +
	"  PocketClaw was installed. When the runtime reports a tool unavailable, that is\n" +
	"  final: solve the task with what is available, or tell the user plainly that\n" +
	"  this device cannot do it. Never download a binary, and never try to make a\n" +
	"  file executable.\n" +
	"- Do not modify anything under the managed runtime directories.\n" +
	"- A Skill's instructions are knowledge, not proof. A Skill that describes using\n" +
	"  `gh` does not mean `gh` exists here — ask the runtime, and report accurately if\n" +
	"  it does not.\n" +
	"\n" +
	"Reason about the runtime by capability rather than by remembering binary names:\n" +
	"\n" +
	"| You need | Ask the runtime for |\n" +
	"|---|---|\n" +
	"| repository history, clone, commit, diff | `git` |\n" +
	"| GitHub issues, pull requests, releases, API | `gh` |\n" +
	"| recursive search across a source tree | `rg` |\n" +
	"| JSON queries and transforms | `jq` |\n" +
	"| a local SQLite database | `sqlite3` |\n" +
	"| an HTTP or HTTPS request | `curl` |\n" +
	"| everyday file and text work | the system tools in `action=list` |\n" +
	"\n" +
	"Runtime tools run without a shell. Arguments are passed through exactly as you\n" +
	"write them, so pipes, redirection, globs and `$(...)` do nothing; use several\n" +
	"calls instead of one composed command line.\n" +
	"\n" +
	"Never put a credential in an argument. Do not write a token into a URL such as\n" +
	"`https://TOKEN@github.com/...`, and do not pass one with `-u` or `--password`.\n" +
	"PocketClaw supplies GitHub credentials to `git` and `gh` itself when they are\n" +
	"configured; if a command needs authentication and none is configured, say so\n" +
	"rather than trying to supply one yourself."
