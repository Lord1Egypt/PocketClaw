import { IconLoader2 } from "@tabler/icons-react"
import type React from "react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { type ModelInfo, deleteModel } from "@/api/models"
import { applyGatewayConfigIfRequired } from "@/lib/restart-required"
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

interface DeleteModelDialogProps {
  model: ModelInfo | null
  onClose: () => void
  onDeleted: () => void
}

export function DeleteModelDialog({
  model,
  onClose,
  onDeleted,
}: DeleteModelDialogProps) {
  const { t } = useTranslation()
  const [deleting, setDeleting] = useState(false)

  // The default model is the one Chat routes through, so removing it is
  // refused rather than performed. It used to be refused by closing the dialog
  // and doing nothing at all: the user pressed Delete, the dialog went away and
  // the model was still there, with no reason given.
  const isDefault = model?.is_default === true

  // Radix closes the dialog on the action button's own click. A delete that
  // fails must not disappear along with the dialog, so the close is taken over
  // here and only performed once the delete has actually succeeded.
  const handleConfirm = async (event: React.MouseEvent) => {
    event.preventDefault()
    if (!model || isDefault) return

    setDeleting(true)
    try {
      await deleteModel(model.index)
      onDeleted()
      // Removal is a configuration change like any other: a gateway still
      // holding the deleted entry would go on routing to it.
      await applyGatewayConfigIfRequired(t, {
        savedMessage: t("models.delete.success", { name: model.model_name }),
        name: model.model_name,
      })
      onClose()
    } catch (e) {
      // A swallowed failure left the model in the list with no explanation and
      // no way to tell a failed delete from a stale list.
      toast.error(e instanceof Error ? e.message : t("models.delete.error"))
    } finally {
      setDeleting(false)
    }
  }

  return (
    <AlertDialog open={model !== null} onOpenChange={(v) => !v && onClose()}>
      <AlertDialogContent size="sm">
        <AlertDialogHeader>
          <AlertDialogTitle>{t("models.delete.title")}</AlertDialogTitle>
          <AlertDialogDescription>
            {isDefault
              ? t("models.delete.defaultBlocked", { name: model?.model_name })
              : t("models.delete.description", { name: model?.model_name })}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onClose} disabled={deleting}>
            {isDefault ? t("common.close") : t("common.cancel")}
          </AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={handleConfirm}
            disabled={deleting || isDefault}
          >
            {deleting && <IconLoader2 className="size-4 animate-spin" />}
            {t("models.delete.confirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
