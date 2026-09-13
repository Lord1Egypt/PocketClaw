import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { type ModelInfo, getModels, setDefaultModel } from "@/api/models"
import {
  isConfiguredEntry,
  isRoutableEntry,
} from "@/lib/configured-model-source"
import { applyGatewayConfigIfRequired } from "@/lib/restart-required"

interface UseChatModelsOptions {
  isConnected: boolean
}

function isLocalModel(model: ModelInfo): boolean {
  const isLocalHostBase = Boolean(
    model.api_base?.includes("localhost") ||
    model.api_base?.includes("127.0.0.1"),
  )

  return (
    model.auth_method === "local" || (!model.auth_method && isLocalHostBase)
  )
}

export function useChatModels({ isConnected }: UseChatModelsOptions) {
  const { t } = useTranslation()
  const [modelList, setModelList] = useState<ModelInfo[]>([])
  const [persistedModelName, setPersistedModelName] = useState("")
  /**
   * The model the user just picked, while the save is still in flight.
   *
   * PC-DEF-042. The selector is fully controlled by server state, and that
   * state was only updated after POST /api/models/default, a second GET, and a
   * gateway restart had all resolved. Until then the trigger went on showing
   * the previous model with nothing to say it was working, so the selection
   * read as having been ignored -- and the user went to Settings and back,
   * which remounts this hook and re-reads the persisted default.
   *
   * Holding the pending choice separately is what lets the trigger answer
   * immediately without ever claiming a save that has not happened: it is
   * cleared on success, when the persisted value takes over, and on failure,
   * when the previous value comes back alongside the error.
   */
  const [pendingModelName, setPendingModelName] = useState("")
  const [settingDefault, setSettingDefault] = useState(false)
  const setDefaultRequestIdRef = useRef(0)

  // What the trigger shows. A pending choice wins, because it is the most
  // recent thing the user actually did.
  const defaultModelName = pendingModelName || persistedModelName

  const syncDefaultModelName = useCallback(
    (models: ModelInfo[], defaultModel: string) => {
      if (models.some((m) => m.model_name === defaultModel)) {
        setPersistedModelName(defaultModel)
        return
      }
      // The configured default is gone -- the model was deleted, or its
      // provider was removed. Chat must not go on pointing at it.
      setPersistedModelName("")
    },
    [],
  )

  const loadModels = useCallback(async () => {
    try {
      const data = await getModels()
      setModelList(data.models)
      syncDefaultModelName(data.models, data.default_model)
    } catch {
      // silently fail
    }
  }, [syncDefaultModelName])

  useEffect(() => {
    const timerId = setTimeout(() => {
      void loadModels()
    }, 0)

    return () => clearTimeout(timerId)
  }, [isConnected, loadModels])

  const handleSetDefault = useCallback(
    async (modelName: string) => {
      if (modelName === defaultModelName) return
      const requestId = ++setDefaultRequestIdRef.current

      // Shown before the first request goes out. Nothing about this claims the
      // model is saved; it says which model the UI is working on.
      setPendingModelName(modelName)
      setSettingDefault(true)

      try {
        await setDefaultModel(modelName)
        const data = await getModels()
        if (requestId !== setDefaultRequestIdRef.current) {
          return
        }

        setModelList(data.models)
        syncDefaultModelName(data.models, data.default_model)
        // Persisted is now the truth, so the pending value must stop shadowing
        // it -- otherwise a later reload could not correct a selection.
        setPendingModelName("")
        // Switching model from the chat header should make that model answer,
        // not leave the user to press Restart Gateway first. This is the slow
        // part, and the selection is already visible and already saved by the
        // time it runs.
        await applyGatewayConfigIfRequired(t, {
          savedMessage: t("models.defaultChangeSuccess"),
          name: modelName,
        })
      } catch (err) {
        console.error("Failed to set default model:", err)
        if (requestId === setDefaultRequestIdRef.current) {
          // The save failed, so the selection did not happen. Showing it as
          // though it had would be the worse half of this defect.
          setPendingModelName("")
        }
        toast.error(err instanceof Error ? err.message : t("models.loadError"))
      } finally {
        if (requestId === setDefaultRequestIdRef.current) {
          setSettingDefault(false)
        }
      }
    },
    [defaultModelName, syncDefaultModelName, t],
  )

  // Selectability comes from the shared source, so the Default selector and the
  // Fallback picker cannot disagree about what counts as a configured model.
  // The shipped keyless provider templates fail `isConfiguredEntry` and never
  // reach any selector.
  const defaultSelectableModels = useMemo(
    () => modelList.filter((m) => isRoutableEntry(m) && isConfiguredEntry(m)),
    [modelList],
  )

  const hasAvailableModels = useMemo(
    () => defaultSelectableModels.some((m) => m.available),
    [defaultSelectableModels],
  )

  const oauthModels = useMemo(
    () =>
      defaultSelectableModels.filter(
        (m) => m.available && m.auth_method === "oauth",
      ),
    [defaultSelectableModels],
  )

  const localModels = useMemo(
    () => defaultSelectableModels.filter((m) => m.available && isLocalModel(m)),
    [defaultSelectableModels],
  )

  const apiKeyModels = useMemo(
    () =>
      defaultSelectableModels.filter(
        (m) => m.available && m.auth_method !== "oauth" && !isLocalModel(m),
      ),
    [defaultSelectableModels],
  )

  return {
    defaultModelName,
    hasAvailableModels,
    apiKeyModels,
    oauthModels,
    localModels,
    settingDefault,
    handleSetDefault,
  }
}
