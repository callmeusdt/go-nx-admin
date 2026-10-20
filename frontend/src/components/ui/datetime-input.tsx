import React, { useEffect, useRef } from 'react'
import flatpickr from 'flatpickr'
import { CalendarDays } from 'lucide-react'
import { Mandarin } from 'flatpickr/dist/l10n/zh'
import { MandarinTraditional } from 'flatpickr/dist/l10n/zh-tw'
import { Spanish } from 'flatpickr/dist/l10n/es'
import { Vietnamese } from 'flatpickr/dist/l10n/vn'
import { useI18n } from '../../contexts/i18n-context'
import type { CustomLocale } from 'flatpickr/dist/types/locale'

const Urdu: CustomLocale = {
  weekdays: {
    shorthand: ['اتوار', 'پیر', 'منگل', 'بدھ', 'جمعرات', 'جمعہ', 'ہفتہ'],
    longhand: ['اتوار', 'پیر', 'منگل', 'بدھ', 'جمعرات', 'جمعہ', 'ہفتہ'],
  },
  months: {
    shorthand: ['جنوری', 'فروری', 'مارچ', 'اپریل', 'مئی', 'جون', 'جولائی', 'اگست', 'ستمبر', 'اکتوبر', 'نومبر', 'دسمبر'],
    longhand: ['جنوری', 'فروری', 'مارچ', 'اپریل', 'مئی', 'جون', 'جولائی', 'اگست', 'ستمبر', 'اکتوبر', 'نومبر', 'دسمبر'],
  },
  rangeSeparator: ' تا ', weekAbbreviation: 'ہفتہ', scrollTitle: 'تبدیل کرنے کے لیے اسکرول کریں',
  toggleTitle: 'تبدیل کرنے کے لیے کلک کریں', yearAriaLabel: 'سال', monthAriaLabel: 'مہینہ',
  hourAriaLabel: 'گھنٹہ', minuteAriaLabel: 'منٹ', time_24hr: true,
}
const calendarLocales = { 'zh-CN': Mandarin, 'zh-TW': MandarinTraditional, es: Spanish, vi: Vietnamese, ur: Urdu }

interface DateTimeInputProps {
  value: string
  placeholder: string
  onChange: (value: string) => void
}

export const DateTimeInput: React.FC<DateTimeInputProps> = ({ value, placeholder, onChange }) => {
  const { locale } = useI18n()
  const ref = useRef<HTMLInputElement>(null)
  const fpRef = useRef<flatpickr.Instance | null>(null)
  const changeRef = useRef(onChange)
  changeRef.current = onChange

  useEffect(() => {
    if (!ref.current) return
    fpRef.current?.destroy()
    fpRef.current = flatpickr(ref.current, {
      enableTime: true,
      dateFormat: 'Y-m-d H:i',
      time_24hr: true,
      allowInput: false,
      locale: calendarLocales[locale as keyof typeof calendarLocales] ?? 'default',
      onChange: (selectedDates, dateStr) => {
        changeRef.current(dateStr)
      },
    })
    return () => { fpRef.current?.destroy() }
  }, [locale])

  useEffect(() => {
    if (!fpRef.current) return
    fpRef.current.setDate(value || '', false)
  }, [value, locale])

  return (
    <div className="relative">
      <input ref={ref} type="text" placeholder={placeholder} readOnly className="flex h-10 w-full rounded-md border border-gray-300 bg-white pl-3 pr-9 py-2 text-sm cursor-pointer hover:border-blue-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
      <CalendarDays className="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400 pointer-events-none" />
    </div>
  )
}
