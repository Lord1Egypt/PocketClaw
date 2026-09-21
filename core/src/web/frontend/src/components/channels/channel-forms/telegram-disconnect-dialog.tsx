import { IconTrash } from "@tabler/icons-react"
import { useCallback, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { disconnectTelegram } from "@/api/telegram-lifecycle"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"

/**
 * Removing the configured Telegram bot.
 *
 * PC-DEF-062. Desktop could pair a bot and then not undo it: the connected card
 * offered Open chat and Reconnect, both of which need the Android host, so the
 * owner had to pick the phone up to remove a bot the browser had created.
 *
 * The confirmation is explicit and spells out what is cleared, because this is
 * destructive and silent otherwise: the token, the owner allowlist and the
 * running channel all go. Core does all of it in one authoritative call — this
 * component never edits configuration fields itself.
 */
interface TelegramDisconnectDialogProps {
  /** Called after removal, so the page reloads its configuration. */
  onDisconnected: () => void
}

export function TelegramDisconnectDialog({
  onDisconnected,
}: TelegramDisconnectDialogProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)

  const confirm = useCallback(async () => {
    setBusy(true)
    try {
      const result = await disconnectTelegram()
      if (result.pending) {
        // Removed from the configuration, but the running channel outlives it
        // until the gateway is idle. PC-DEF-030's rule, and not a failure —
        // saying "removed" flatly would be the dishonest half of it.
        toast.info(t("channels.telegram.disconnect.removedPending"))
      } else {
        toast.success(t("channels.telegram.disconnect.removed"))
      }
      setOpen(false)
      onDisconnected()
    } catch {
      toast.error(t("channels.telegram.disconnect.failed"))
    } finally {
      setBusy(false)
    }
  }, [onDisconnected, t])

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" className="min-h-10">
          <IconTrash className="size-4" />
          {t("channels.telegram.disconnect.action")}
        </Button>
      </DialogTrigger>
      <DialogContent data-testid="telegram-disconnect-dialog">
        <DialogHeader>
          <DialogTitle>{t("channels.telegram.disconnect.title")}</DialogTitle>
          <DialogDescription>
            {t("channels.telegram.disconnect.body")}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="ghost" className="min-h-10" disabled={busy}>
              {t("common.cancel")}
            </Button>
          </DialogClose>
          <Button
            variant="destructive"
            className="min-h-10"
            disabled={busy}
            onClick={() => void confirm()}
          >
            {t("channels.telegram.disconnect.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
