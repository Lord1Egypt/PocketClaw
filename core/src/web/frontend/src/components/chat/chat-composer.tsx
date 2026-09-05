import { IconArrowUp, IconPhotoPlus, IconX } from "@tabler/icons-react"
import {
  type ClipboardEvent as ReactClipboardEvent,
  type DragEvent as ReactDragEvent,
  type KeyboardEvent as ReactKeyboardEvent,
  useRef,
} from "react"
import { useTranslation } from "react-i18next"
import TextareaAutosize from "react-textarea-autosize"

import { ContextUsageRing } from "@/components/chat/context-usage-ring"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import type { ChatAttachment, ContextUsage } from "@/store/chat"

export type ChatInputDisabledReason =
  | "gatewayUnknown"
  | "gatewayStarting"
  | "gatewayRestarting"
  | "gatewayStopping"
  | "gatewayStopped"
  | "gatewayError"
  | "websocketConnecting"
  | "websocketDisconnected"
  | "websocketError"
  | "noDefaultModel"

interface ChatComposerProps {
  input: string
  attachments: ChatAttachment[]
  onInputChange: (value: string) => void
  onAddImages: () => void
  onPaste: (event: ReactClipboardEvent<HTMLTextAreaElement>) => void
  onDragEnter: (event: ReactDragEvent<HTMLDivElement>) => void
  onDragLeave: (event: ReactDragEvent<HTMLDivElement>) => void
  onDragOver: (event: ReactDragEvent<HTMLDivElement>) => void
  onDrop: (event: ReactDragEvent<HTMLDivElement>) => void
  onRemoveAttachment: (index: number) => void
  onSend: () => void
  onContextDetail?: () => void
  inputDisabledReason: ChatInputDisabledReason | null
  // Specific, already-sanitized reason from the launcher. Replaces the generic
  // message when the backend knows exactly why the gateway is unusable.
  inputDisabledDetail?: string
  canSend: boolean
  isDragActive: boolean
  contextUsage?: ContextUsage
}

export function ChatComposer({
  input,
  attachments,
  onInputChange,
  onAddImages,
  onPaste,
  onDragEnter,
  onDragLeave,
  onDragOver,
  onDrop,
  onRemoveAttachment,
  onSend,
  onContextDetail,
  inputDisabledReason,
  inputDisabledDetail,
  canSend,
  isDragActive,
  contextUsage,
}: ChatComposerProps) {
  const { t } = useTranslation()
  const canInput = inputDisabledReason === null
  const composingRef = useRef(false)
  const hasInput = input.trim().length > 0
  const disabledMessage =
    inputDisabledReason === null
      ? null
      : (inputDisabledDetail ??
        t(`chat.disabledPlaceholder.${inputDisabledReason}`))
  const placeholder = disabledMessage ?? t("chat.placeholder")

  const handleKeyDown = (e: ReactKeyboardEvent<HTMLTextAreaElement>) => {
    const nativeEvent = e.nativeEvent as Event & {
      isComposing?: boolean
      keyCode?: number
    }
    if (
      composingRef.current ||
      nativeEvent.isComposing ||
      nativeEvent.keyCode === 229
    ) {
      return
    }
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      onSend()
    }
  }

  return (
    <div className="before:bg-background pointer-events-none relative z-10 -mt-[24px] shrink-0 [scrollbar-gutter:stable] overflow-y-auto px-4 pb-[calc(0.75rem+env(safe-area-inset-bottom))] before:pointer-events-none before:absolute before:inset-x-0 before:top-[24px] before:bottom-0 before:content-[''] md:px-8 md:pb-6">
      <div className="pointer-events-auto mx-auto flex w-full max-w-[78ch] flex-col items-stretch">
        <div
          className={cn(
            "bg-pc-surface-2 border-pc-line focus-within:border-pc-claw focus-within:ring-pc-claw-line relative flex w-full flex-col rounded-2xl border p-3 transition-colors focus-within:ring-2",
            isDragActive && "border-pc-claw bg-pc-claw-soft",
          )}
          onDragEnter={onDragEnter}
          onDragLeave={onDragLeave}
          onDragOver={onDragOver}
          onDrop={onDrop}
        >
          {isDragActive && (
            <div className="border-pc-claw bg-pc-claw-soft pointer-events-none absolute inset-0 z-10 flex items-center justify-center rounded-2xl border-2 border-dashed">
              <div className="bg-pc-surface-1 text-pc-text rounded-full px-4 py-2 text-sm font-medium">
                {t("chat.dropImagesActive")}
              </div>
            </div>
          )}

          {attachments.length > 0 && (
            <div className="mb-3 flex flex-wrap gap-2 px-2">
              {attachments.map((attachment, index) => (
                <div
                  key={`${attachment.url}-${index}`}
                  className="bg-pc-surface-1 border-pc-line relative h-20 w-20 overflow-hidden rounded-xl border"
                >
                  <img
                    src={attachment.url}
                    alt={attachment.filename || t("chat.uploadedImage")}
                    className="h-full w-full object-cover"
                  />
                  <button
                    type="button"
                    onClick={() => onRemoveAttachment(index)}
                    className="bg-pc-surface-1 text-pc-text border-pc-line hover:bg-pc-surface-3 absolute end-1 top-1 inline-flex h-6 w-6 items-center justify-center rounded-full border transition-colors"
                    aria-label={t("chat.removeImage")}
                    title={t("chat.removeImage")}
                  >
                    <IconX className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          )}

          <TextareaAutosize
            value={input}
            onChange={(e) => onInputChange(e.target.value)}
            onCompositionStart={() => {
              composingRef.current = true
            }}
            onCompositionEnd={() => {
              composingRef.current = false
            }}
            onPaste={onPaste}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            disabled={!canInput}
            title={disabledMessage || undefined}
            className={cn(
              "placeholder:text-pc-faint text-pc-text max-h-[40vh] min-h-[48px] resize-none border-0 bg-transparent px-2 py-1.5 text-[15px] leading-relaxed shadow-none transition-colors focus-visible:ring-0 focus-visible:outline-none dark:bg-transparent",
              !canInput && "cursor-not-allowed",
            )}
            minRows={1}
            maxRows={8}
          />

          <div className="border-t-pc-line/70 -mx-3 -mb-3 mt-2 flex items-center justify-between gap-2 border-t px-2 py-1.5">
            <div className="flex min-w-0 items-center gap-1">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="text-pc-muted hover:text-pc-text hover:bg-pc-surface-3 size-10 rounded-full"
                onClick={onAddImages}
                disabled={!canInput}
                aria-label={t("chat.attachImage")}
                title={t("chat.attachImage")}
              >
                <IconPhotoPlus className="size-5" />
              </Button>
            </div>

            <div className="flex items-center gap-1.5">
              {contextUsage && (
                <ContextUsageRing
                  usage={contextUsage}
                  onDetailClick={onContextDetail}
                />
              )}
              {/* One slot, one footprint. The button is disabled rather
                  than unmounted when input is blocked, so the footer keeps
                  its height and nothing jumps. */}
              <span tabIndex={canInput && !canSend ? 0 : undefined}>
                <Button
                  type="button"
                  size="icon"
                  className="bg-pc-claw text-pc-claw-ink hover:bg-pc-claw-hover size-10 rounded-full transition-colors active:scale-95 disabled:opacity-40"
                  onClick={onSend}
                  disabled={!canInput || !canSend}
                  aria-label={t("chat.sendMessage")}
                >
                  <IconArrowUp className="size-5" />
                </Button>
              </span>
            </div>
          </div>
        </div>

        <div
          aria-hidden={!hasInput}
          className={cn(
            "border-pc-line bg-pc-surface-1 text-pc-faint mx-auto mt-2 inline-flex items-center rounded-md border px-3 py-1 text-[11px] transition-all duration-200",
            hasInput
              ? "translate-y-0 opacity-100"
              : "pointer-events-none -translate-y-1 opacity-0",
          )}
        >
          {t("chat.composeHint")}
        </div>
      </div>
    </div>
  )
}
