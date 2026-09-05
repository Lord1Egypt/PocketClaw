import {
  IconDotsVertical,
  IconLanguage,
  IconLogout,
  IconMoon,
  IconRefresh,
  IconSun,
} from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { LANGUAGE_OPTIONS, selectedLanguageCode } from "@/i18n/languages"

interface HeaderUtilityMenuProps {
  theme: string
  onToggleTheme: () => void
  onLogout: () => void
  onRestartGateway: () => void
  restartRequired: boolean
  restartDisabled: boolean
  className?: string
  /// Test seam: Radix only mounts the item list once the menu is open.
  defaultOpen?: boolean
}

/**
 * The low-frequency header actions, collected behind one control.
 *
 * At phone width the toolbar was carrying a menu button, a mark, a gateway
 * control and four icon buttons in one row, so nothing in it read as more
 * important than anything else. Language, theme and sign-out are things a
 * user touches rarely; gathering them here leaves the narrow toolbar saying
 * only what matters at a glance — menu, identity, runtime state.
 *
 * Nothing is hidden: every action is still reachable, keyboard-navigable and
 * announced. Radix owns the focus contract, so focus returns to this trigger
 * when the menu closes.
 */
export function HeaderUtilityMenu({
  theme,
  onToggleTheme,
  onLogout,
  onRestartGateway,
  restartRequired,
  restartDisabled,
  className,
  defaultOpen,
}: HeaderUtilityMenuProps) {
  const { i18n, t } = useTranslation()
  const current = selectedLanguageCode(i18n.language ?? i18n.resolvedLanguage)
  const themeLabel = theme === "dark" ? t("common.lightMode") : t("common.darkMode")

  return (
    <DropdownMenu defaultOpen={defaultOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className={className}
          aria-label={t("header.moreActions")}
        >
          <IconDotsVertical className="size-5" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-52">
        {restartRequired && (
          <>
            <DropdownMenuItem
              disabled={restartDisabled}
              onSelect={onRestartGateway}
            >
              <IconRefresh className="size-4" />
              {t("header.gateway.action.restart")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
          </>
        )}

        <DropdownMenuItem onSelect={onToggleTheme}>
          {theme === "dark" ? (
            <IconSun className="size-4" />
          ) : (
            <IconMoon className="size-4" />
          )}
          {themeLabel}
        </DropdownMenuItem>

        <DropdownMenuSub>
          <DropdownMenuSubTrigger>
            <IconLanguage className="size-4" />
            {t("common.language")}
          </DropdownMenuSubTrigger>
          <DropdownMenuSubContent className="max-h-[60vh] overflow-y-auto">
            <DropdownMenuRadioGroup
              value={current}
              onValueChange={(code) => void i18n.changeLanguage(code)}
            >
              {LANGUAGE_OPTIONS.map((option) => (
                <DropdownMenuRadioItem
                  key={option.code}
                  value={option.code}
                  // Endonyms keep their own script and direction regardless of
                  // the console's current one.
                  lang={option.code}
                  dir={i18n.dir(option.code)}
                >
                  {option.endonym}
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuSubContent>
        </DropdownMenuSub>

        <DropdownMenuSeparator />

        <DropdownMenuItem variant="destructive" onSelect={onLogout}>
          <IconLogout className="size-4" />
          {t("header.logout.tooltip")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
