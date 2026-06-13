import React, { useEffect, useRef } from 'react'
import flatpickr from 'flatpickr'
import { CalendarDays } from 'lucide-react'

interface DateTimeInputProps {
  value: string
  placeholder: string
  onChange: (value: string) => void
}

export const DateTimeInput: React.FC<DateTimeInputProps> = ({ value, placeholder, onChange }) => {
  const ref = useRef<HTMLInputElement>(null)
  const fpRef = useRef<flatpickr.Instance | null>(null)

  useEffect(() => {
    if (!ref.current) return
    fpRef.current?.destroy()
    fpRef.current = flatpickr(ref.current, {
      enableTime: true,
      dateFormat: 'Y-m-d H:i',
      time_24hr: true,
      allowInput: false,
      onChange: (selectedDates, dateStr) => {
        onChange(dateStr)
      },
    })
    return () => { fpRef.current?.destroy() }
  }, [])

  useEffect(() => {
    if (!fpRef.current) return
    fpRef.current.setDate(value || '', false)
  }, [value])

  return (
    <div className="relative">
      <input ref={ref} type="text" placeholder={placeholder} readOnly className="flex h-10 w-full rounded-md border border-gray-300 bg-white pl-3 pr-9 py-2 text-sm cursor-pointer hover:border-blue-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
      <CalendarDays className="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400 pointer-events-none" />
    </div>
  )
}
