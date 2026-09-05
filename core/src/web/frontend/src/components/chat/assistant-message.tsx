import {
  IconBrain,
  IconCheck,
  IconChevronDown,
  IconCopy,
  IconDownload,
  IconFileText,
  IconTool,
} from "@tabler/icons-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import ReactMarkdown from "react-markdown"
import rehypeHighlight from "rehype-highlight"
import rehypeRaw from "rehype-raw"
import rehypeSanitize from "rehype-sanitize"
import remarkGfm from "remark-gfm"

import {
  MarkdownCodeBlock,
  MessageCodeBlock,
} from "@/components/chat/message-code-block"
import { Button } from "@/components/ui/button"
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard"
import { formatMessageTime } from "@/hooks/use-pico-chat"
import { PocketClawMark } from "@/components/brand/pocketclaw-mark"
import { cn } from "@/lib/utils"
import {
  type AssistantMessageKind,
  type ChatAttachment,
  type ChatToolCall,
} from "@/store/chat"

interface AssistantMessageProps {
  content: string
  attachments?: ChatAttachment[]
  kind?: AssistantMessageKind
  modelName?: string
  toolCalls?: ChatToolCall[]
  timestamp?: string | number
}

export function AssistantMessage({
  content,
  attachments = [],
  kind = "normal",
  modelName,
  toolCalls = [],
  timestamp = "",
}: AssistantMessageProps) {
  const { t } = useTranslation()
  const { copy, isCopied } = useCopyToClipboard()
  const isThought = kind === "thought"
  const isToolCalls = kind === "tool_calls"
  const isCollapsedBlock = isThought || isToolCalls
  const hasText = content.trim().length > 0
  const hasToolCalls = toolCalls.length > 0
  const imageAttachments = attachments.filter(
    (attachment) => attachment.type === "image",
  )
  const fileAttachments = attachments.filter(
    (attachment) => attachment.type !== "image",
  )
  const [isExpanded, setIsExpanded] = useState(true)
  const formattedTimestamp =
    timestamp !== "" ? formatMessageTime(timestamp) : ""
  const collapsedLabel = isThought
    ? t("chat.reasoningLabel")
    : t("chat.toolCallsLabel")
  const copyMessageLabel = isCopied
    ? t("chat.copiedLabel")
    : t("chat.copyMessage")
  const trimmedModelName = modelName?.trim() ?? ""

  return (
    <div className="group flex w-full flex-col gap-1.5">
      {!isCollapsedBlock && (
        <div className="text-pc-faint flex items-center justify-between gap-2 text-xs">
          <div className="flex items-center gap-2 ps-4">
            <PocketClawMark className="text-pc-claw size-3.5" />
            <span className="font-medium">PocketClaw</span>
            {trimmedModelName && (
              <>
                <span aria-hidden="true">·</span>
                <span className="pc-mono">{trimmedModelName}</span>
              </>
            )}
            {formattedTimestamp && (
              <>
                <span aria-hidden="true">·</span>
                <span>{formattedTimestamp}</span>
              </>
            )}
          </div>
        </div>
      )}

      {(hasText || isCollapsedBlock || hasToolCalls) && (
        <div
          className={cn(
            "relative",
            isCollapsedBlock
              ? "bg-pc-surface-1 border-pc-line text-pc-muted overflow-hidden rounded-md border"
              : "pc-rail text-pc-text ps-4",
          )}
        >
          {isCollapsedBlock && (
            <div
              className="text-pc-faint hover:text-pc-muted flex cursor-pointer items-center justify-between px-3 py-2 text-[12px] font-medium transition-colors select-none"
              onClick={() => setIsExpanded(!isExpanded)}
            >
              <div className="flex items-center gap-1.5">
                {isThought ? (
                  <IconBrain className="size-3.5" />
                ) : (
                  <IconTool className="size-3.5" />
                )}
                <span>{collapsedLabel}</span>
                {trimmedModelName && (
                  <span className="pc-mono text-pc-faint">
                    {trimmedModelName}
                  </span>
                )}
              </div>
              <div className="flex items-center gap-2">
                {formattedTimestamp && (
                  <span className="opacity-50">{formattedTimestamp}</span>
                )}
                <IconChevronDown
                  className={cn(
                    "size-3.5 opacity-0 transition-all duration-200 group-hover:opacity-100",
                    isExpanded ? "rotate-180" : "",
                  )}
                />
              </div>
            </div>
          )}
          {(!isCollapsedBlock || isExpanded) && isToolCalls && hasToolCalls && (
            <div className="space-y-3 px-3 pt-0 pb-3">
              {toolCalls.map((toolCall, index) => {
                const explanation =
                  toolCall.extraContent?.toolFeedbackExplanation?.trim() ?? ""
                const toolName = toolCall.function?.name?.trim() ?? ""
                const toolArguments = toolCall.function?.arguments?.trim() ?? ""
                const hasFunctionSummary = toolName || toolArguments

                if (!explanation && !hasFunctionSummary) {
                  return null
                }

                return (
                  <div
                    key={toolCall.id ?? `${toolName}-${index}`}
                    className={cn(
                      "space-y-3",
                      index > 0 && "border-pc-line border-t pt-3",
                    )}
                  >
                    {explanation && (
                      <div className="space-y-1.5">
                        <div className="text-pc-faint pc-micro">
                          {t("chat.toolCallExplanationLabel")}
                        </div>
                        <div className="prose dark:prose-invert prose-p:my-1.5 prose-p:whitespace-pre-wrap max-w-none text-[13px] leading-relaxed [overflow-wrap:anywhere] break-words opacity-75">
                          <ReactMarkdown
                            remarkPlugins={[remarkGfm]}
                            rehypePlugins={[
                              rehypeRaw,
                              rehypeSanitize,
                              rehypeHighlight,
                            ]}
                            components={{
                              pre: MarkdownCodeBlock,
                            }}
                          >
                            {explanation}
                          </ReactMarkdown>
                        </div>
                      </div>
                    )}

                    {hasFunctionSummary && (
                      <div
                        className={cn(
                          "space-y-1.5",
                          explanation && "border-pc-line border-t pt-3",
                        )}
                      >
                        <div className="text-pc-faint pc-micro">
                          {t("chat.toolCallFunctionLabel")}
                        </div>
                        <div className="bg-pc-canvas border-pc-line space-y-2 rounded-md border px-3 py-2.5">
                          {toolName && !toolArguments && (
                            <div className="text-pc-text pc-mono text-[12px] font-semibold">
                              {toolName}
                            </div>
                          )}
                          {toolArguments && (
                            <MessageCodeBlock
                              code={toolArguments}
                              language="json"
                              label={
                                toolName || t("chat.toolCallArgumentsLabel")
                              }
                              className="my-0 shadow-none"
                              bodyClassName="px-3 py-2 text-[12px] leading-relaxed"
                            />
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
          {(!isCollapsedBlock || isExpanded) && !isToolCalls && hasText && (
            <div
              className={cn(
                "prose dark:prose-invert prose-pre:my-2 prose-pre:overflow-x-auto prose-pre:rounded-lg prose-pre:border prose-pre:bg-pc-surface-1 prose-pre:border-pc-line prose-pre:p-0 max-w-none [overflow-wrap:anywhere] break-words",
                isThought
                  ? "prose-p:my-1.5 prose-p:whitespace-pre-wrap px-3 pt-0 pb-3 text-[13px] leading-relaxed opacity-70"
                  : "prose-p:my-2 prose-p:whitespace-pre-wrap p-4 text-[15px] leading-relaxed",
              )}
            >
              <ReactMarkdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeRaw, rehypeSanitize, rehypeHighlight]}
                components={{
                  pre: MarkdownCodeBlock,
                }}
              >
                {content}
              </ReactMarkdown>
            </div>
          )}

          {!isCollapsedBlock && hasText && (
            <Button
              variant="ghost"
              size="icon"
              className="bg-pc-surface-1 hover:bg-pc-surface-3 absolute end-0 top-0 h-7 w-7 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
              onClick={() => void copy(content)}
              aria-label={copyMessageLabel}
              title={copyMessageLabel}
            >
              {isCopied ? (
                <IconCheck className="text-pc-success h-4 w-4" />
              ) : (
                <IconCopy className="text-pc-muted h-4 w-4" />
              )}
            </Button>
          )}
        </div>
      )}

      {imageAttachments.length > 0 && (
        <div className="mt-1 flex flex-wrap gap-2">
          {imageAttachments.map((attachment, index) => (
            <a
              key={`${attachment.url}-${index}`}
              href={attachment.url}
              target="_blank"
              rel="noreferrer"
              className="group/img border-pc-line bg-pc-surface-1 hover:border-pc-claw/50 relative overflow-hidden rounded-xl border transition-colors"
            >
              <img
                src={attachment.url}
                alt={attachment.filename || t("chat.uploadedImage")}
                className="max-h-80 max-w-[280px] object-contain transition-transform duration-300 group-hover/img:scale-[1.02]"
              />
              
            </a>
          ))}
        </div>
      )}

      {fileAttachments.length > 0 && (
        <div className="mt-1 flex flex-wrap gap-3">
          {fileAttachments.map((attachment, index) => (
            <a
              key={`${attachment.url}-${index}`}
              href={attachment.url}
              download={attachment.filename}
              className="group/file border-pc-line bg-pc-surface-1 hover:border-pc-claw/50 flex w-fit max-w-sm min-w-[220px] items-center gap-3.5 rounded-xl border px-4 py-3 transition-colors"
            >
              <div className="bg-pc-surface-3 text-pc-claw flex h-10 w-10 shrink-0 items-center justify-center rounded-lg">
                <IconFileText className="h-5 w-5" />
              </div>
              <div className="flex min-w-0 flex-1 flex-col pr-1">
                <span className="text-pc-text group-hover/file:text-pc-claw truncate text-[14px] leading-tight font-medium transition-colors">
                  {attachment.filename || t("chat.downloadFile")}
                </span>
                <span className="text-pc-faint pc-micro mt-1">
                  {attachment.filename?.split(".").pop()?.toUpperCase() ||
                    "FILE"}
                </span>
              </div>
              <div className="bg-pc-surface-3 text-pc-muted group-hover/file:bg-pc-claw group-hover/file:text-pc-claw-ink flex h-8 w-8 shrink-0 items-center justify-center rounded-full transition-colors">
                <IconDownload className="h-4 w-4" />
              </div>
            </a>
          ))}
        </div>
      )}
    </div>
  )
}
