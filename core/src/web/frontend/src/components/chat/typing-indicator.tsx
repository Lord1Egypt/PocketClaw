import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

export function TypingIndicator() {
  const { t } = useTranslation()
  const thinkingSteps = [
    t("chat.thinking.step1"),
    t("chat.thinking.step2"),
    t("chat.thinking.step3"),
    t("chat.thinking.step4"),
  ]
  const [stepIndex, setStepIndex] = useState(0)

  useEffect(() => {
    const stepsCount = thinkingSteps.length
    const interval = setInterval(() => {
      setStepIndex((prev) => (prev + 1) % stepsCount)
    }, 3000)
    return () => clearInterval(interval)
  }, [thinkingSteps.length])

  return (
    <div
      className="flex w-full flex-col gap-1.5"
      role="status"
      aria-live="polite"
    >
      {/* The same rail the answer will arrive on, so nothing shifts sideways
          when the placeholder is replaced by the real message. */}
      <div className="pc-rail flex flex-col gap-2.5 ps-4">
        <div className="text-pc-faint flex items-center gap-2 text-xs">
          <span
            aria-hidden="true"
            className="bg-pc-signal pc-signal-pulse size-2 shrink-0 rounded-full"
          />
          <span className="font-medium">PocketClaw</span>
        </div>

        <div className="bg-pc-surface-3 relative h-1 w-36 overflow-hidden rounded-full">
          <div className="from-pc-signal/30 via-pc-signal to-pc-signal/30 absolute inset-0 animate-[shimmer_2s_infinite] rounded-full bg-gradient-to-r bg-[length:200%_100%]" />
        </div>

        <p
          key={stepIndex}
          className="text-pc-muted animate-[fadeSlideIn_0.4s_ease-out] text-sm"
        >
          {thinkingSteps[stepIndex]}
        </p>
      </div>
    </div>
  )
}
