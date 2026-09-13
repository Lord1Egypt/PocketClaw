import { IconLoader2 } from "@tabler/icons-react"
import type React from "react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { type ProviderInfo, deleteProvider } from "@/api/providers"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { applyGatewayConfigIfRequired } from "@/lib/restart-required"

interface DeleteProviderDialogProps {
  /** The provider to delete, or null when the dialog is closed. */
  provider: ProviderInfo | null
  /** Display name for the provider. */
  label: string
  onClose: () => void
  onDeleted: () => void
}

/**
 * Confirms deleting a whole provider.
 *
 * Deleting a provider deletes its models: a model whose provider is gone has no
 * endpoint and no credential. That is a bigger action than deleting one model,
 * so the dependent models are counted and named before it happens, and the
 * default model is called out separately — losing it means Chat has no model
 * until another is chosen.
 */
export function DeleteProviderDialog({
  provider,
  label,
  onClose,
  onDeleted,
}: DeleteProviderDialogProps) {
  const { t } = useTranslation()
  const [deleting, setDeleting] = useState(false)

  // Radix closes the dialog on the action button's own click. A delete that
  // fails must not disappear along with the dialog, so the close is taken over
  // here and only performed once the delete has actually succeeded.
  const handleConfirm = async (event: React.MouseEvent) => {
    event.preventDefault()
    if (!provider) return

    setDeleting(true)
    try {
      await deleteProvider(provider.provider)
      onDeleted()
      // The running gateway still holds the deleted models and their
      // credentials. Applying the change is what makes the removal real.
      await applyGatewayConfigIfRequired(t, {
        savedMessage: t("models.provider.deleteSuccess", { name: label }),
        name: label,
      })
    } catch (e) {
      // A swallowed failure would leave the provider listed with no explanation
      // and no way to tell a failed delete from a stale list.
      toast.error(
        e instanceof Error ? e.message : t("models.provider.deleteError"),
      )
    } finally {
      setDeleting(false)
    }
  }

  return (
    <AlertDialog open={provider !== null} onOpenChange={(v) => !v && onClose()}>
      <AlertDialogContent size="sm">
        <AlertDialogHeader>
          <AlertDialogTitle>
            {t("models.provider.deleteTitle", { name: label })}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("models.provider.deleteDescription", {
              count: provider?.model_count ?? 0,
            })}
          </AlertDialogDescription>
        </AlertDialogHeader>

        {provider && provider.models.length > 0 && (
          <ul className="text-muted-foreground max-h-40 space-y-1 overflow-y-auto text-sm">
            {provider.models.map((model) => (
              <li key={model.index} className="truncate">
                {model.model_name}
                {model.is_default && (
                  <span className="text-pc-warning ms-1.5">
                    {t("models.provider.deleteDefaultNote")}
                  </span>
                )}
              </li>
            ))}
          </ul>
        )}

        <AlertDialogFooter>
          <AlertDialogCancel onClick={onClose} disabled={deleting}>
            {t("common.cancel")}
          </AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={handleConfirm}
            disabled={deleting}
          >
            {deleting && <IconLoader2 className="size-4 animate-spin" />}
            {t("models.provider.deleteConfirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
