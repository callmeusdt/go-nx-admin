import React, { useState } from 'react'
import { createRoot } from 'react-dom/client'
import { DateTimeInput } from '../src/components/ui/datetime-input'
import { I18nProvider } from '../src/contexts/i18n-context'
import 'flatpickr/dist/flatpickr.min.css'

function Fixture() {
  const [date, setDate] = useState('2026-10-20')
  const [datetime, setDatetime] = useState('2026-10-20 12:34')
  return <I18nProvider options={{ defaultLocale: 'en-US' }}>
    <DateTimeInput mode="date" placeholder="Business date" value={date} onChange={setDate} />
    <DateTimeInput placeholder="Datetime" value={datetime} onChange={setDatetime} />
    <output data-testid="date">{date}</output><output data-testid="datetime">{datetime}</output>
    <button onClick={() => setDate('2026-10-21')}>Replace</button>
    <button onClick={() => setDate('')}>Clear</button>
  </I18nProvider>
}
createRoot(document.getElementById('root')!).render(<Fixture />)
