const ALLOW_FROM_HIDDEN_CHARS_RE =
  /\u200b|\u200c|\u200d|\u200e|\u200f|\u202a|\u202b|\u202c|\u202d|\u202e|\u2060|\u2061|\u2062|\u2063|\u2064|\u2066|\u2067|\u2068|\u2069|\ufeff/g

function normalizeStringListItems(
  items: string[],
  options: { stripHiddenChars?: boolean } = {},
): string[] {
  const result: string[] = []
  const seen = new Set<string>()

  for (const item of items) {
    const normalized = options.stripHiddenChars
      ? item.replace(ALLOW_FROM_HIDDEN_CHARS_RE, "")
      : item
    const trimmed = normalized.trim()
    if (trimmed.length === 0 || seen.has(trimmed)) {
      continue
    }
    seen.add(trimmed)
    result.push(trimmed)
  }

  return result
}

function splitStringList(
  raw: string,
  separators: RegExp,
  options: { stripHiddenChars?: boolean } = {},
): string[] {
  if (raw.trim() === "") {
    return []
  }
  return normalizeStringListItems(raw.split(separators), options)
}

export function asStringArray(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.filter((item): item is string => typeof item === "string")
  }
  // Legacy shape: before list fields were saved as arrays, the console joined
  // them with "\n" into a single string. Split on exactly that separator —
  // nothing else was ever a separator in a stored value, so a stored entry
  // containing a comma or semicolon stays intact.
  if (typeof value === "string") {
    return normalizeStringListItems(value.split(/\r?\n/))
  }
  return []
}

export function parseAllowFromInput(raw: string): string[] {
  return splitStringList(raw, /[,\uFF0C、;；\n\r\t]+/, {
    stripHiddenChars: true,
  })
}

export function parseConservativeStringListInput(raw: string): string[] {
  return splitStringList(raw, /[,\uFF0C\n\r\t]+/)
}

export function normalizeAllowFromValues(value: unknown): string[] {
  return normalizeStringListItems(asStringArray(value), {
    stripHiddenChars: true,
  })
}

export function mergeUniqueStringItems(
  currentItems: string[],
  nextItems: string[],
): string[] {
  return normalizeStringListItems([...currentItems, ...nextItems])
}

// serializeStringArrayForSubmit shapes a list field for the save payload.
//
// Canonical shape is a JSON array. It used to join entries with "\n", which the
// backend could not read back: `FlexibleStringSlice` turns "a\nb" into the
// single entry "a\nb" rather than two, and the plain []string fields
// (group_trigger.prefixes, allow_origins, onebot group_trigger_prefix) reject a
// bare string outright. A legacy string still loads — see asStringArray — so an
// old config is repaired the next time the user saves it.
export function serializeStringArrayForSubmit(value: unknown): unknown {
  if (typeof value === "string") {
    return value
  }
  if (!Array.isArray(value)) {
    return value
  }
  return normalizeStringListItems(asStringArray(value))
}
