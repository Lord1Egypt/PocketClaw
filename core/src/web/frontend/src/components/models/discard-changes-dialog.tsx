import { useTranslation } from "react-i18next"

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

interface DiscardChangesDialogProps {
  open: boolean
  onKeepEditing: () => void
  onDiscard: () => void
}

/**
 * Asked before an edit is thrown away.
 *
 * A provider form closes on the overlay, on Escape and on Cancel, and every one
 * of those used to drop whatever had been typed without saying so. A new API
 * key is exactly the kind of thing a user types once and does not want to type
 * again, so losing it silently is not an acceptable close.
 */
export function DiscardChangesDialog({
  open,
  onKeepEditing,
  onDiscard,
}: DiscardChangesDialogProps) {
  const { t } = useTranslation()

  return (
    <AlertDialog open={open} onOpenChange={(v) => !v && onKeepEditing()}>
      <AlertDialogContent size="sm">
        <AlertDialogHeader>
          <AlertDialogTitle>{t("models.discard.title")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("models.discard.description")}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={onKeepEditing}>
            {t("models.discard.keepEditing")}
          </AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={onDiscard}>
            {t("models.discard.confirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
