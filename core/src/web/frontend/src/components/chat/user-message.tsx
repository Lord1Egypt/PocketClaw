import { IconCheck, IconCopy } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard"
import { formatMessageTime } from "@/hooks/use-pico-chat"
import { cn } from "@/lib/utils"
import type { ChatAttachment } from "@/store/chat"

interface UserMessageProps {
  content: string
  attachments?: ChatAttachment[]
  timestamp?: string | number
}

/**
 * The user's turn: a contained bubble on the inline-end edge.
 *
 * Only this half of the conversation is a bubble. The assistant's turn is set
 * as a document, so the thread has a rhythm instead of two symmetrical columns
 * facing each other.
 */
export function UserMessage({
  content,
  attachments = [],
  timestamp = "",
}: UserMessageProps) {
  const { t } = useTranslation()
  const { copy, isCopied } = useCopyToClipboard()
  const hasText = content.trim().length > 0
  const isCommand = content.trim().startsWith("/")
  const imageAttachments = attachments.filter(
    (attachment) => attachment.type === "image",
  )
  const copyMessageLabel = isCopied
    ? t("chat.copiedLabel")
    : t("chat.copyMessage")
  const formattedTimestamp =
    timestamp !== "" ? formatMessageTime(timestamp) : ""

  return (
    <div className="group flex w-full flex-col items-end gap-1.5">
      {imageAttachments.length > 0 && (
        <div className="flex max-w-[78%] flex-wrap justify-end gap-2">
          {imageAttachments.map((attachment, index) => (
            <img
              key={`${attachment.url}-${index}`}
              src={attachment.url}
              alt={attachment.filename || t("chat.uploadedImage")}
              className="border-pc-line max-h-72 max-w-full rounded-xl border object-cover"
            />
          ))}
        </div>
      )}

      {hasText && (
        <div className="relative max-w-[78%]">
          <div
            className={cn(
              "wrap-break-word border-pc-line rounded-2xl border px-4 py-3 whitespace-pre-wrap",
              // A slash command is a machine value, so it is set in mono and
              // reads as a readout rather than as speech.
              isCommand
                ? "bg-pc-surface-1 pc-mono text-pc-text rounded-ee-md text-[14px]"
                : "bg-pc-surface-2 text-pc-text rounded-ee-md text-[15px] leading-relaxed",
            )}
          >
            {isCommand ? (
              <div className="flex items-start gap-2.5">
                <span className="text-pc-claw font-bold select-none">❯</span>
                <span className="mt-[1px]">{content}</span>
              </div>
            ) : (
              content
            )}
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="bg-pc-surface-1 hover:bg-pc-surface-3 text-pc-muted absolute end-2 top-2 h-7 w-7 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
            onClick={() => void copy(content)}
            aria-label={copyMessageLabel}
            title={copyMessageLabel}
          >
            {isCopied ? (
              <IconCheck className="text-pc-success h-4 w-4" />
            ) : (
              <IconCopy className="h-4 w-4" />
            )}
          </Button>
        </div>
      )}

      {formattedTimestamp && (
        <span className="text-pc-faint px-1 text-[12px]">
          {formattedTimestamp}
        </span>
      )}
    </div>
  )
}
