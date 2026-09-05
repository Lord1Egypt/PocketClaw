/**
 * One source, three selectors.
 *
 * The reported defect was a Fallback picker offering Azure, Cerebras, Groq,
 * Ollama and Volcengine to a user who had configured only OpenCode and Gemini.
 * The cause was two selectors applying two different filters to the same list —
 * the chat Default selector checked `available`, the Fallback picker did not.
 *
 * These are structural guards. They do not re-test the filtering logic, which
 * `configured-model-source.test.ts` covers; they assert that no routing
 * selector grows its own catalog again.
 */
import fsSync from "node:fs"
import path from "node:path"

import { describe, expect, it } from "vitest"

const read = (file: string) =>
  fsSync.readFileSync(path.join(process.cwd(), file), "utf8")

const SHARED_SOURCE = "src/lib/configured-model-source.ts"
const DISCOVERY_HOOK = "src/hooks/use-configured-models.ts"

/// Every surface that lets a user assign a routing role.
const ROUTING_SELECTORS = [
  "src/components/models/fallback-models-section.tsx",
  "src/components/models/default-model-section.tsx",
  "src/hooks/use-chat-models.ts",
]

describe("every routing selector draws from the shared source", () => {
  it.each(ROUTING_SELECTORS)("%s imports the shared source", (file) => {
    const source = read(file)
    expect(source).toMatch(/configured-model-source|use-configured-models/)
  })

  it.each(ROUTING_SELECTORS)(
    "%s never reaches for the global provider preset list",
    (file) => {
      const source = read(file)
      // `provider_options` is the global preset registry. A selector may accept
      // it to resolve display names and fetch support, but it must not read
      // `common_models` — that is the hard-coded per-provider model list that
      // put unconfigured providers in the picker.
      expect(source).not.toContain("common_models")
      expect(source).not.toContain("getProviderCatalogMap")
      expect(source).not.toContain("provider-registry")
    },
  )

  it.each(ROUTING_SELECTORS)(
    "%s does not re-implement the configured filter itself",
    (file) => {
      const source = read(file)
      // The two ways a selector used to decide this on its own. Both now live
      // in the shared source, and a local copy is how the two drifted apart.
      expect(source).not.toMatch(/status\s*!==\s*["']unconfigured["']/)
      expect(source).not.toMatch(/\.filter\([^)]*\bm\.available\b/)
    },
  )
})

describe("the shared source is the only place the rule lives", () => {
  const source = read(SHARED_SOURCE)

  it("decides configured-ness from the backend status, not from a key field", () => {
    expect(source).toContain('model.status !== "unconfigured"')
    // Reading api_key from the browser would be both wrong and useless — it is
    // masked in the list response.
    expect(source).not.toContain("api_key")
  })

  it("keys a model by provider instance and id, never by id alone", () => {
    expect(source).toContain("normalizeProvider(provider)")
    expect(source).toContain("normalizeApiBase(apiBase)")
  })
})

describe("discovery never carries a credential", () => {
  const hook = read(DISCOVERY_HOOK)

  it("sends an index, so the backend resolves the stored key itself", () => {
    expect(hook).toContain("model_index")
    expect(hook).not.toContain("api_key")
  })

  it("isolates one provider's failure with allSettled", () => {
    // `Promise.all` would abandon the remaining providers on the first
    // rejection, which is exactly the "one failure empties the picker" defect.
    expect(hook).toContain("Promise.allSettled")
    expect(hook).not.toMatch(/Promise\.all\(/)
  })
})

describe("materialization goes through the transactional endpoint", () => {
  it.each([
    "src/components/models/fallback-models-section.tsx",
    "src/components/models/default-model-section.tsx",
  ])("%s uses materializeModel rather than addModel", (file) => {
    const source = read(file)
    expect(source).toContain("materializeModel")
    // addModel would create the entry in a separate call, reopening the
    // half-configured window the single endpoint exists to close.
    expect(source).not.toMatch(/\baddModel\b/)
  })

  it("assigns the default role in the same call that creates the entry", () => {
    const source = read("src/components/models/default-model-section.tsx")
    expect(source).toContain('role: "default"')
  })
})
