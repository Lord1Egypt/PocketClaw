import wrapAnsi from "wrap-ansi"

const OSC_PATTERN = new RegExp(
  String.raw`(?:\u001B\]|\u009D)[\s\S]*?(?:\u0007|\u001B\\|\u009C)`,
  "g",
)
const UNTERMINATED_OSC_PATTERN = new RegExp(
  String.raw`(?:\u001B\]|\u009D)[^\n]*$`,
  "gm",
)
const STRING_CONTROL_PATTERN = new RegExp(
  String.raw`(?:\u001B[P^_X]|[\u0090\u0098\u009E\u009F])[\s\S]*?(?:\u001B\\|\u009C)`,
  "g",
)
const UNTERMINATED_STRING_CONTROL_PATTERN = new RegExp(
  String.raw`(?:\u001B[P^_X]|[\u0090\u0098\u009E\u009F])[^\n]*$`,
  "gm",
)
const CSI_PATTERN = new RegExp(
  String.raw`(?:\u001B\[|\u009B)[0-?]*[ -/]*[@-~]`,
  "g",
)
const ORPHANED_CSI_PATTERN =
  /\[(?:\??[0-9:;<=>]+)[0-9:;<=>? ]*[ABCDEFGHJKSTfmnsu]/g
const OTHER_ESCAPE_PATTERN = new RegExp(String.raw`\u001B[ -/]*[@-~]`, "g")
const ROUTINE_PICO_WS_PATTERN =
  /(?:^| > )GET \/pico\/ws (?:101|2[0-9]{2})(?:\s|$)/
const PICO_WS_REQUEST_PATTERN = /((?:^| > )[A-Z]+) \/pico\/ws ([0-9]{3})(\s|$)/g
const LEGACY_GATEWAY_START_PATTERN = /Starting gateway process \([^\r\n)]*\)/g
const PICO_LOGGER_COMPONENT_PATTERN =
  /(^|[ \t])([A-Z]{3}) pico ([^ \t]+:[0-9]+)([ \t]+>)/gm
const PICO_LOGGER_CALLER_PATTERN =
  /(^|[ \t])([A-Z]{3}) ([^ \t]+) pico\.go:([0-9]+)([ \t]+>)/gm

/**
 * Browser-side enforcement of the shared PocketClaw user-visible log
 * contract. The backend applies the same contract before API storage; this is
 * deliberately idempotent so raw or legacy API lines cannot bypass it.
 */
export function normalizeUserVisibleLog(input: string): string {
  if (!input) return input

  const text = input
    .replaceAll("\r\n", "\n")
    .replace(OSC_PATTERN, "")
    .replace(UNTERMINATED_OSC_PATTERN, "")
    .replace(STRING_CONTROL_PATTERN, "")
    .replace(UNTERMINATED_STRING_CONTROL_PATTERN, "")
    .replace(CSI_PATTERN, "")
    .replace(ORPHANED_CSI_PATTERN, "")
    .replace(OTHER_ESCAPE_PATTERN, "")

  let output = ""
  let currentLine: string[] = []

  const flushLine = (newline: boolean) => {
    output += currentLine.join("")
    currentLine = []
    if (newline) output += "\n"
  }

  for (const character of text) {
    const codePoint = character.codePointAt(0)!
    if (character === "\n") {
      flushLine(true)
    } else if (character === "\r") {
      currentLine = []
    } else if (character === "\b") {
      currentLine.pop()
    } else if (character === "\t") {
      currentLine.push(character)
    } else if (codePoint < 0x20 || (codePoint >= 0x7f && codePoint <= 0x9f)) {
      continue
    } else {
      currentLine.push(character)
    }
  }
  flushLine(false)

  let result = output.replace(
    LEGACY_GATEWAY_START_PATTERN,
    "Starting gateway process",
  )
  result = result.replace(PICO_LOGGER_COMPONENT_PATTERN, "$1$2 realtime $3$4")
  result = result.replace(
    PICO_LOGGER_CALLER_PATTERN,
    "$1$2 $3 realtime.go:$4$5",
  )
  if (ROUTINE_PICO_WS_PATTERN.test(result)) return ""
  result = result.replace(
    PICO_WS_REQUEST_PATTERN,
    "$1 /internal realtime connection $2$3",
  )
  return result
}

export function wrapPlainTextLogLine(line: string, columns: number): string {
  if (columns < 20) return line

  return wrapAnsi(line, columns, {
    hard: true,
    trim: false,
    wordWrap: false,
  })
}
