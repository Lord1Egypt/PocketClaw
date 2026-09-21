import { IconLoader2, IconTrash } from "@tabler/icons-react"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import type { ModelProviderOption } from "@/api/models"
import {
  type ProviderInfo,
  getProvider,
  updateProvider,
} from "@/api/providers"
import { maskedSecretPlaceholder } from "@/components/secret-placeholder"
import { Field, KeyInput } from "@/components/shared-form"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { applyGatewayConfigIfRequired } from "@/lib/restart-required"

import { DeleteProviderDialog } from "./delete-provider-dialog"
import { ProviderIcon } from "./provider-icon"
import {
  type ProviderCatalogEntry,
  getProviderCatalogMap,
} from "./provider-registry"

interface ManageProviderSheetProps {
  /** Canonical provider key, or null when the sheet is closed. */
  provider: string | null
  providerOptions?: ModelProviderOption[]
  onClose: () => void
  /** Called after any change that the model list must be reloaded for. */
  onChanged: () => void
}

/**
 * Provider-scoped management: what belongs to the provider rather than to one
 * of its models.
 *
 * The provider owns the endpoint and the credential; a model owns its model ID,
 * its default flag and its own overrides. Keeping that split is what makes
 * rotating one key a single action instead of an edit per model.
 *
 * Every control is a labelled, full-height touch target and the primary action
 * is in the header as well as the footer: this sheet is used on a phone, where a
 * footer action disappears under the soft keyboard the moment the credential
 * field is focused, and a tooltip explains nothing at all.
 */
export function ManageProviderSheet({
  provider,
  providerOptions,
  onClose,
  onChanged,
}: ManageProviderSheetProps) {
  const { t } = useTranslation()
  const [info, setInfo] = useState<ProviderInfo | null>(null)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  // null is "no error". An entry carries the thrown message when there is one
  // and otherwise names which operation failed, so the text is resolved at
  // render time and loading never has to depend on the translator.
  const [error, setError] = useState<{
    message?: string
    fallbackKey: string
  } | null>(null)
  const [apiKey, setApiKey] = useState("")
  const [apiBase, setApiBase] = useState("")
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [reloadToken, setReloadToken] = useState(0)

  const catalog: ProviderCatalogEntry | undefined = provider
    ? getProviderCatalogMap(providerOptions).get(provider)
    : undefined

  // Depends on the provider and an explicit reload token, and on nothing that
  // changes identity between renders. An earlier version loaded through a
  // callback that closed over the translator, so every render re-ran the load
  // and cleared the key the user was typing.
  useEffect(() => {
    if (!provider) {
      setInfo(null)
      setApiKey("")
      setApiBase("")
      setError(null)
      return
    }

    let cancelled = false
    setLoading(true)
    setError(null)
    getProvider(provider)
      .then((loaded) => {
        if (cancelled) return
        setInfo(loaded)
        setApiBase(loaded.api_base ?? "")
        // Never prefilled: the stored credential does not leave the backend in
        // the clear, and an empty field is what "leave it alone" looks like.
        setApiKey("")
      })
      .catch((e: unknown) => {
        if (cancelled) return
        setError({
          message: e instanceof Error ? e.message : undefined,
          fallbackKey: "models.provider.loadError",
        })
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    // A provider switched while a response was in flight must not be overwritten
    // by the response for the one before it.
    return () => {
      cancelled = true
    }
  }, [provider, reloadToken])

  const baseEditable = info ? !info.api_base_mixed : false
  const baseChanged = baseEditable && apiBase.trim() !== (info?.api_base ?? "")
  const keyChanged = apiKey.trim() !== ""
  const isDirty = keyChanged || baseChanged

  const handleSave = async () => {
    if (!provider || !info || !isDirty) return
    setSaving(true)
    setError(null)
    try {
      // Only what changed is sent. An omitted api_key means "leave the stored
      // credential alone", so saving a base URL never blanks the key.
      const result = await updateProvider(provider, {
        ...(keyChanged ? { api_key: apiKey.trim() } : {}),
        ...(baseChanged ? { api_base: apiBase.trim() } : {}),
      })
      onChanged()
      // A rotated credential is only rotated once the gateway is running on it.
      // The running process holds the old key until it restarts, so applying the
      // change is the difference between the key being stored and being used.
      await applyGatewayConfigIfRequired(t, {
        savedMessage: keyChanged
          ? t("models.provider.keyRotated", { count: result.models_updated })
          : t("models.provider.saved"),
        name: catalog?.label ?? provider,
      })
      setReloadToken((token) => token + 1)
    } catch (e) {
      setError({
        message: e instanceof Error ? e.message : undefined,
        fallbackKey: "models.provider.saveError",
      })
    } finally {
      setSaving(false)
    }
  }

  const credentialSummary = () => {
    if (!info) return ""
    switch (info.credential_state) {
      case "shared":
        return t("models.provider.credentialShared", {
          key: maskedSecretPlaceholder(
            info.api_key_masked ?? "",
            t("models.provider.credentialSetGeneric"),
          ),
        })
      case "mixed":
        return t("models.provider.credentialMixed")
      default:
        return t("models.provider.credentialUnset")
    }
  }

  return (
    <>
      <Sheet
        open={provider !== null}
        onOpenChange={(v) => !v && !saving && onClose()}
      >
        <SheetContent
          side="right"
          className="flex flex-col gap-0 p-0 data-[side=right]:!w-full data-[side=right]:sm:!w-[560px] data-[side=right]:sm:!max-w-[560px]"
        >
          {/* The primary action sits in the header as well as the footer. On a
              phone the soft keyboard covers the footer as soon as the API key
              field is focused, which is precisely when there is something to
              save. The end padding leaves room for the sheet's close button. */}
          <SheetHeader className="border-b-muted border-b px-6 py-5 pe-16">
            <div className="flex items-start justify-between gap-3">
              <div className="flex min-w-0 items-start gap-2.5">
                {catalog && <ProviderIcon provider={catalog} size="md" />}
                <div className="min-w-0">
                  <SheetTitle className="text-base">
                    {t("models.provider.manageTitle", {
                      name: catalog?.label ?? provider,
                    })}
                  </SheetTitle>
                  <SheetDescription className="text-xs">
                    {info
                      ? t("models.provider.modelCount", {
                          count: info.model_count,
                        })
                      : t("models.provider.loading")}
                  </SheetDescription>
                </div>
              </div>
              <Button
                size="sm"
                className="min-h-10 shrink-0"
                onClick={handleSave}
                disabled={!isDirty || saving || loading}
              >
                {saving && <IconLoader2 className="size-4 animate-spin" />}
                {t("models.provider.save")}
              </Button>
            </div>
          </SheetHeader>

          <div className="min-h-0 flex-1 overflow-y-auto">
            <div className="space-y-5 px-6 py-5">
              {loading && !info && (
                <p className="text-muted-foreground flex items-center gap-2 text-sm">
                  <IconLoader2 className="size-4 animate-spin" />
                  {t("models.provider.loading")}
                </p>
              )}

              {error && (
                <p className="text-destructive text-sm" role="alert">
                  {error.message ?? t(error.fallbackKey)}
                </p>
              )}

              {info && (
                <>
                  <Field
                    label={t("models.provider.credential")}
                    hint={credentialSummary()}
                  >
                    <KeyInput
                      value={apiKey}
                      onChange={setApiKey}
                      placeholder={t("models.provider.replaceKeyPlaceholder")}
                    />
                    <p className="text-muted-foreground text-xs">
                      {t("models.provider.replaceKeyHint", {
                        count: info.model_count,
                      })}
                    </p>
                  </Field>

                  <Field
                    label={t("models.field.apiBase")}
                    hint={
                      baseEditable
                        ? t("models.provider.apiBaseHint")
                        : t("models.provider.apiBaseMixed")
                    }
                  >
                    <Input
                      value={baseEditable ? apiBase : ""}
                      onChange={(event) => setApiBase(event.target.value)}
                      disabled={!baseEditable}
                      placeholder={
                        baseEditable ? (catalog?.defaultApiBase ?? "") : ""
                      }
                      className="font-mono text-sm"
                    />
                  </Field>

                  <section className="space-y-2">
                    <h3 className="text-sm font-medium">
                      {t("models.provider.modelsHeading")}
                    </h3>
                    {/* The provider's models, named. A delete confirmation that
                        says "2 models" without saying which two asks the user to
                        remember what they configured. */}
                    <ul className="divide-border divide-y rounded-lg border">
                      {info.models.map((model) => (
                        <li
                          key={model.index}
                          className="flex items-center justify-between gap-3 px-3 py-2.5"
                        >
                          <span className="min-w-0">
                            <span className="block truncate text-sm">
                              {model.model_name}
                            </span>
                            <span className="text-muted-foreground block truncate font-mono text-xs">
                              {model.model}
                            </span>
                          </span>
                          {model.is_default && (
                            <span className="text-pc-faint pc-micro shrink-0">
                              {t("models.badge.default")}
                            </span>
                          )}
                        </li>
                      ))}
                    </ul>
                  </section>
                </>
              )}
            </div>
          </div>

          {info && (
            <div className="border-t-muted flex flex-col gap-2 border-t px-6 py-4">
              <Button
                variant="destructive"
                className="min-h-10 w-full"
                onClick={() => setDeleteOpen(true)}
                disabled={saving}
              >
                <IconTrash className="size-4" />
                {t("models.provider.delete")}
              </Button>
              <Button
                variant="outline"
                className="min-h-10 w-full"
                onClick={onClose}
                disabled={saving}
              >
                {t("common.close")}
              </Button>
            </div>
          )}
        </SheetContent>
      </Sheet>

      <DeleteProviderDialog
        provider={deleteOpen ? info : null}
        label={catalog?.label ?? provider ?? ""}
        onClose={() => setDeleteOpen(false)}
        onDeleted={() => {
          setDeleteOpen(false)
          onChanged()
          onClose()
          toast.success(
            t("models.provider.deleteSuccess", {
              name: catalog?.label ?? provider,
            }),
          )
        }}
      />
    </>
  )
}
