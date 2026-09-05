/**
 * Live model discovery across the providers the user has actually configured.
 *
 * Discovery goes through `POST /api/models/fetch` with a `model_index` and no
 * key: the backend looks the credential up from stored config after checking
 * the provider and API base match, so the secret never reaches the browser.
 *
 * Each provider instance discovers independently. One provider timing out is
 * one group showing a retry, not an empty picker — and never a reason to fall
 * back to the global provider catalog, which is the defect this whole module
 * exists to fix.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from "react"

import {
  type ModelInfo,
  type ModelProviderOption,
  fetchUpstreamModels,
} from "@/api/models"
import {
  type ConfiguredProviderGroup,
  type DiscoveryState,
  buildConfiguredProviderGroups,
  isConfiguredEntry,
  isRoutableEntry,
  providerInstanceKey,
} from "@/lib/configured-model-source"

interface UseConfiguredModelsOptions {
  models: ModelInfo[]
  providerOptions?: ModelProviderOption[]
  defaultModelName?: string
  /**
   * Whether to discover as soon as the provider instances are known. Off by
   * default: a picker that is never opened should not call four upstream APIs.
   */
  autoDiscover?: boolean
}

export interface UseConfiguredModelsResult {
  groups: ConfiguredProviderGroup[]
  /** Runs discovery for one provider instance. Safe to call repeatedly. */
  discover: (groupKey: string) => Promise<void>
  /** Runs discovery for every fetchable provider instance, independently. */
  discoverAll: () => Promise<void>
  discovering: boolean
}

export function useConfiguredModels({
  models,
  providerOptions,
  defaultModelName,
  autoDiscover = false,
}: UseConfiguredModelsOptions): UseConfiguredModelsResult {
  const [discovered, setDiscovered] = useState<Record<string, string[]>>({})
  const [discoveryState, setDiscoveryState] = useState<
    Record<string, DiscoveryState>
  >({})

  // Survives unmount so a late response cannot set state on a dead component.
  const aliveRef = useRef(true)
  useEffect(() => {
    aliveRef.current = true
    return () => {
      aliveRef.current = false
    }
  }, [])

  const groups = useMemo(
    () =>
      buildConfiguredProviderGroups({
        models,
        providerOptions,
        discovered,
        discoveryState,
        defaultModelName,
      }),
    [models, providerOptions, discovered, discoveryState, defaultModelName],
  )

  /**
   * The provider instances worth querying, resolved from the same rules the
   * source module uses so the two cannot disagree about what is configured.
   */
  const fetchable = useMemo(() => {
    const seen = new Map<
      string,
      { provider: string; apiBase: string; sourceIndex: number }
    >()
    for (const model of models) {
      if (!isConfiguredEntry(model) || !isRoutableEntry(model)) continue
      const provider = model.provider ?? ""
      const option = providerOptions?.find(
        (candidate) =>
          candidate.id.trim().toLowerCase() === provider.trim().toLowerCase(),
      )
      if (option?.supports_fetch !== true) continue
      const key = providerInstanceKey(provider, model.api_base)
      if (seen.has(key)) continue
      seen.set(key, {
        provider,
        apiBase: model.api_base ?? "",
        // The backend's index into model_list, which is what it resolves the
        // stored credential from.
        sourceIndex: model.index,
      })
    }
    return seen
  }, [models, providerOptions])

  const discover = useCallback(
    async (groupKey: string) => {
      const target = fetchable.get(groupKey)
      if (!target) return

      setDiscoveryState((prev) => ({ ...prev, [groupKey]: { phase: "loading" } }))
      try {
        const res = await fetchUpstreamModels({
          provider: target.provider,
          api_base: target.apiBase || undefined,
          // An index, not a key. The backend resolves the credential itself.
          model_index: target.sourceIndex,
        })
        if (!aliveRef.current) return
        setDiscovered((prev) => ({
          ...prev,
          [groupKey]: (res.models ?? []).map((entry) => entry.id),
        }))
        setDiscoveryState((prev) => ({
          ...prev,
          [groupKey]: { phase: "ready", fetchedAt: Date.now() },
        }))
      } catch (e) {
        if (!aliveRef.current) return
        // One provider's failure is that provider's problem. The group keeps
        // whatever it already had, which is at minimum its configured entries.
        setDiscoveryState((prev) => ({
          ...prev,
          [groupKey]: {
            phase: "error",
            message: e instanceof Error ? e.message : String(e),
          },
        }))
      }
    },
    [fetchable],
  )

  const discoverAll = useCallback(async () => {
    // allSettled, not all: one rejection must not cancel the others.
    await Promise.allSettled([...fetchable.keys()].map((key) => discover(key)))
  }, [fetchable, discover])

  const autoDiscoveredRef = useRef<string>("")
  useEffect(() => {
    if (!autoDiscover) return
    const signature = [...fetchable.keys()].sort().join("|")
    if (signature === "" || signature === autoDiscoveredRef.current) return
    autoDiscoveredRef.current = signature
    void discoverAll()
  }, [autoDiscover, fetchable, discoverAll])

  const discovering = useMemo(
    () =>
      Object.values(discoveryState).some((state) => state.phase === "loading"),
    [discoveryState],
  )

  return { groups, discover, discoverAll, discovering }
}
