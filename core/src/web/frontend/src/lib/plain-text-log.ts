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
  /\[(?:\?[0-9:;]+|[0-9][0-9:;]*)[ ]*[ABCDEFGHJKSTfmnsu]/g
const OTHER_ESCAPE_PATTERN = new RegExp(String.raw`\u001B[ -/]*[@-~]`, "g")
const ROUTINE_PICO_WS_PATTERN =
  /(?:^| > )GET \/pico\/ws (?:101|2[0-9]{2})(?:\s|$)/
const PICO_WS_REQUEST_PATTERN = /((?:^| > )[A-Z]+) \/pico\/ws ([0-9]{3})(\s|$)/g
const LEGACY_GATEWAY_START_PATTERN = /Starting gateway process \([^\r\n)]*\)/g
const PICO_LOGGER_COMPONENT_PATTERN =
  /(^|[ \t])([A-Z]{3}) pico ([^ \t]+:[0-9]+)([ \t]+>)/gm
const PICO_LOGGER_CALLER_PATTERN =
  /(^|[ \t])([A-Z]{3}) ([^ \t]+) (?:pico|pocketclaw)\.go:([0-9]+)([ \t]+>)/gm
const TELEGRAM_BOT_API_URL_PATTERN =
  /https?:\/\/[^\s"']*\/bot[^/\s"']+\/(?:test\/)?([A-Za-z][A-Za-z0-9_]*)/gi
const TELEGRAM_API_CALL_WRAPPER_PATTERN =
  /API call to: "Telegram API call: ([A-Za-z][A-Za-z0-9_]*)"/gi
const AUTHORIZATION_CREDENTIAL_PATTERN =
  /(authorization[=:][ \t]*)(?:\[?(?:bearer|basic)[ \t]+)[A-Za-z0-9._~+/%:=-]+\]?/gi
const LEGACY_PID_FILE_PATH_PATTERN =
  /(?:[^\s"']*[\\/])?\.picoclaw\.pid(?:\.tmp)?([\s"']|$)/g
const TELEGRAM_SUCCESSFUL_NIL_ERROR =
  /((?:^| > )API response [A-Za-z][A-Za-z0-9_]*: Ok: true, Err:) \[<nil>\]/g
const ROUTINE_TELEGRAM_GET_UPDATES_CALL =
  /(?:^| > )Telegram API call: getUpdates(?:, with data:.*)?$/
const ROUTINE_EMPTY_GET_UPDATES_RESPONSE =
  /(?:^| > )API response getUpdates: Ok: true, Err: none, Result: \[\](?:\s|$)/
const SENSITIVE_SESSION_FIELD_PATTERN =
  /(^|[ \t])(session_key|scope_key|route_main_session)=(?:"(?:\\.|[^"\\])*"|[^ \t\r\n]*)/gm
const INTERNAL_IDENTITY_FIELD_PATTERN =
  /(^|[ \t])(chat_id|inbound_chat_id|target_chat_id|sender_id|inbound_sender_id|user_id|session_id|connection_id|conn_id|runtime_id)=(?:"(?:\\.|[^"\\])*"|[^ \t\r\n]*)/gm
const RAW_CONTENT_FIELD_PATTERN =
  /(^|[ \t])(arguments|args|content|messages_json|payload|preview|prompt|reasoning|response|text|tools_json)=(?:"(?:\\.|[^"\\])*"|[^ \t\r\n]*)/gm
const TELEGRAM_API_RESPONSE_PATTERN =
  /^API response ([A-Za-z][A-Za-z0-9_]*): Ok: (true|false), Err: \[([\s\S]*?)\](?:, Result: ([\s\S]*))?$/
const SAFE_TELEGRAM_API_CALL_PATTERN =
  /^Telegram API call: ([A-Za-z][A-Za-z0-9_]*)(?:, with data:.*)?$/i
const TELEGRAM_UPDATE_TYPES = [
  "message",
  "edited_message",
  "channel_post",
  "edited_channel_post",
  "business_connection",
  "business_message",
  "edited_business_message",
  "deleted_business_messages",
  "message_reaction",
  "message_reaction_count",
  "inline_query",
  "chosen_inline_result",
  "callback_query",
  "shipping_query",
  "pre_checkout_query",
  "purchased_paid_media",
  "poll",
  "poll_answer",
  "my_chat_member",
  "chat_member",
  "chat_join_request",
  "chat_boost",
  "removed_chat_boost",
] as const
const EXACT_COMPATIBILITY_MESSAGES = new Map([
  ["Starting Pico Protocol channel", "Starting PocketClaw realtime channel"],
  ["Pico Protocol channel started", "PocketClaw realtime channel started"],
  ["Stopping Pico Protocol channel", "Stopping PocketClaw realtime channel"],
  ["Pico Protocol channel stopped", "PocketClaw realtime channel stopped"],
  [
    "wrote pid file: <gateway PID file> success",
    "Gateway PID file written successfully",
  ],
  [
    "Failed to finalize streamed pico reasoning",
    "Failed to finalize streamed realtime reasoning",
  ],
  [
    "Pico reasoning publish skipped (timeout/cancel)",
    "Realtime reasoning publish skipped (timeout/cancel)",
  ],
  [
    "Failed to publish pico reasoning (best-effort)",
    "Failed to publish realtime reasoning (best-effort)",
  ],
  ["Failed to publish pico reasoning", "Failed to publish realtime reasoning"],
  [
    "Failed to publish pico interim assistant content",
    "Failed to publish realtime interim assistant content",
  ],
  [
    "Failed to serialize pico tool calls",
    "Failed to serialize realtime tool calls",
  ],
  [
    "Failed to publish pico tool calls",
    "Failed to publish realtime tool calls",
  ],
])

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
  result = result.replace(TELEGRAM_BOT_API_URL_PATTERN, "Telegram API call: $1")
  result = result.replace(
    TELEGRAM_API_CALL_WRAPPER_PATTERN,
    "Telegram API call: $1",
  )
  result = result.replace(AUTHORIZATION_CREDENTIAL_PATTERN, "$1<redacted>")
  result = result.replace(LEGACY_PID_FILE_PATH_PATTERN, "<gateway PID file>$1")
  result = normalizeTelegramMessageInLine(result)
  if (!result) return ""
  result = result.replace(TELEGRAM_SUCCESSFUL_NIL_ERROR, "$1 none")
  result = normalizePrivateStructuredFields(result)
  result = normalizePicoStructuredFields(result)
  result = normalizeExactCompatibilityMessages(result)
  if (
    ROUTINE_TELEGRAM_GET_UPDATES_CALL.test(result) ||
    ROUTINE_EMPTY_GET_UPDATES_RESPONSE.test(result)
  ) {
    return ""
  }
  if (ROUTINE_PICO_WS_PATTERN.test(result)) return ""
  result = result.replace(
    PICO_WS_REQUEST_PATTERN,
    "$1 /internal realtime connection $2$3",
  )
  return result
}

function normalizeTelegramMessageInLine(input: string): string {
  const separator = input.indexOf(" > ")
  const messageStart = separator < 0 ? 0 : separator + " > ".length
  const prefix = input.slice(0, messageStart)
  const message = input.slice(messageStart)

  const call = message.match(SAFE_TELEGRAM_API_CALL_PATTERN)
  if (call) {
    if (call[1].toLowerCase() === "getupdates") return ""
    return `${prefix}Telegram API call: ${call[1]}`
  }

  const response = message.match(TELEGRAM_API_RESPONSE_PATTERN)
  if (!response) return input
  const [, operation, ok, error, rawResult] = response
  if (ok === "false") {
    const errorCode = error.match(/^([0-9]+)(?:\s|$)/)?.[1]
    return `${prefix}Telegram API failed operation=${operation} ok=false${errorCode ? ` error_code=${errorCode}` : ""}`
  }
  if (operation.toLowerCase() !== "getupdates") {
    return `${prefix}Telegram API completed operation=${operation} ok=true`
  }
  if (rawResult === undefined) {
    return `${prefix}Telegram API response operation=getUpdates ok=true result=malformed`
  }

  try {
    const updates: unknown = JSON.parse(rawResult)
    if (!Array.isArray(updates)) throw new Error("not an update array")
    if (updates.length === 0) return ""
    const types = new Set<string>()
    for (const update of updates) {
      if (!update || typeof update !== "object" || Array.isArray(update)) {
        types.add("unknown")
        continue
      }
      const record = update as Record<string, unknown>
      const updateType = TELEGRAM_UPDATE_TYPES.find(
        (candidate) => record[candidate] !== undefined && record[candidate] !== null,
      )
      types.add(updateType ?? "unknown")
    }
    const sortedTypes = [...types].sort()
    const typeField = sortedTypes.length === 1 ? "type" : "types"
    return `${prefix}Telegram update received updates=${updates.length} ${typeField}=${sortedTypes.join(",")}`
  } catch {
    return `${prefix}Telegram API response operation=getUpdates ok=true result=malformed`
  }
}

function normalizePrivateStructuredFields(input: string): string {
  return input
    .replace(SENSITIVE_SESSION_FIELD_PATTERN, "$1$2=<redacted>")
    .replace(INTERNAL_IDENTITY_FIELD_PATTERN, "$1$2=<internal>")
    .replace(RAW_CONTENT_FIELD_PATTERN, "$1$2=<redacted>")
}

function normalizePicoStructuredFields(input: string): string {
  return input
    .split("\n")
    .map((line) => {
      const internalChannel =
        hasExactLogToken(line, "channel=pocketclaw") ||
        hasExactLogToken(line, "channel=pico")
      let result = line
      for (const field of [
        "channel",
        "inbound_channel",
        "route_channel",
        "scope_channel",
        "target_channel",
      ]) {
        result = replaceExactLogToken(
          result,
          `${field}=pico`,
          `${field}=pocketclaw`,
        )
      }
      result = replaceExactLogToken(result, "type=pico", "type=pocketclaw")
      if (internalChannel) {
        result = replaceExactLogToken(result, "path=/pico/", "path=<internal>")
      }
      return result
    })
    .join("\n")
}

function normalizeExactCompatibilityMessages(input: string): string {
  return input
    .split("\n")
    .map((line) => {
      const separator = line.indexOf(" > ")
      const messageStart = separator < 0 ? 0 : separator + " > ".length
      const messageAndFields = line.slice(messageStart)
      for (const [message, replacement] of EXACT_COMPATIBILITY_MESSAGES) {
        if (messageAndFields === message) {
          return line.slice(0, messageStart) + replacement
        }
        if (messageAndFields.startsWith(`${message} `)) {
          return (
            line.slice(0, messageStart) +
            replacement +
            messageAndFields.slice(message.length)
          )
        }
      }
      return line
    })
    .join("\n")
}

function hasExactLogToken(input: string, token: string): boolean {
  return replaceExactLogToken(input, token, `${token}\0`) !== input
}

function replaceExactLogToken(
  input: string,
  token: string,
  replacement: string,
): string {
  let output = ""
  let searchFrom = 0
  while (searchFrom < input.length) {
    const start = input.indexOf(token, searchFrom)
    if (start < 0) break
    const end = start + token.length
    const beforeOK = start === 0 || isLogTokenSpace(input[start - 1])
    const afterOK = end === input.length || isLogTokenSpace(input[end])
    if (beforeOK && afterOK) {
      output += input.slice(searchFrom, start) + replacement
      searchFrom = end
    } else {
      output += input.slice(searchFrom, start + 1)
      searchFrom = start + 1
    }
  }
  return searchFrom === 0 ? input : output + input.slice(searchFrom)
}

function isLogTokenSpace(value: string): boolean {
  return value === " " || value === "\t" || value === "\r" || value === "\n"
}
