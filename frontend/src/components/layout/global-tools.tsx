import React from 'react'
import { Check, ChevronDown } from 'lucide-react'
import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import { useI18n } from '../../contexts/i18n-context'

export const GlobalTools: React.FC = () => {
  const { locale, setLocale, languages, t } = useI18n()
  const current = languages.find(item => item.locale === locale) || languages[0]

  return (
    <div className="flex items-center gap-1 text-gray-500">
      <DropdownMenu.Root>
        <DropdownMenu.Trigger asChild>
          <button className="inline-flex items-center gap-1.5 px-2 py-1.5 rounded-lg hover:bg-gray-100 text-xs" title={t('common.language')}>
            <span className="text-base leading-none">{current.flag || current.label}</span>
            <ChevronDown className="w-3 h-3" />
          </button>
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content
            align="end"
            sideOffset={8}
            className="z-50 min-w-[150px] rounded-lg border border-gray-200 bg-white p-1 shadow-lg"
          >
            {languages.map(item => (
              <DropdownMenu.Item
                key={item.locale}
                onSelect={() => setLocale(item.locale)}
                className="flex cursor-pointer select-none items-center gap-2 rounded-md px-2.5 py-2 text-sm text-gray-700 outline-none hover:bg-gray-100"
              >
                <span className="text-base leading-none">{item.flag}</span>
                <span className="flex-1">{item.label}</span>
                {locale === item.locale && <Check className="h-3.5 w-3.5 text-blue-500" />}
              </DropdownMenu.Item>
            ))}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </div>
  )
}
