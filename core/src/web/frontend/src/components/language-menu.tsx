import { IconLanguage } from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { LANGUAGE_OPTIONS, selectedLanguageCode } from "@/i18n/languages"

interface LanguageMenuProps {
  variant?: "ghost" | "outline"
  className?: string
  /// Test seam: Radix only mounts the item list once the menu is open.
  defaultOpen?: boolean
}

/**
 * The manual language override.
 *
 * There is exactly one language state in the console — i18next's. Choosing an
 * entry calls `changeLanguage`, which is what persists the choice to
 * localStorage and re-renders every subscriber. The host's `?lng=` still wins
 * on the next load, because i18next reads the query string before storage.
 */
export function LanguageMenu({
  variant = "ghost",
  className,
  defaultOpen,
}: LanguageMenuProps) {
  const { i18n, t } = useTranslation()
  const current = selectedLanguageCode(i18n.language ?? i18n.resolvedLanguage)

  return (
    <DropdownMenu defaultOpen={defaultOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          variant={variant}
          size="icon"
          className={className}
          aria-label={t("common.language")}
        >
          <IconLanguage className="size-4.5" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="max-h-[70vh] overflow-y-auto">
        <DropdownMenuRadioGroup
          value={current}
          onValueChange={(code) => void i18n.changeLanguage(code)}
        >
          {LANGUAGE_OPTIONS.map((option) => (
            <DropdownMenuRadioItem
              key={option.code}
              value={option.code}
              // Endonyms keep their own script and direction regardless of the
              // console's current one.
              lang={option.code}
              dir={i18n.dir(option.code)}
            >
              {option.endonym}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
