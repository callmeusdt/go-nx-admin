import React, { useState } from 'react'
import { createRoot } from 'react-dom/client'
import { RichEditor } from '../src/components/ui/rich-editor'
import { I18nProvider } from '../src/contexts/i18n-context'

function Fixture() {
  const [value, setValue] = useState('<p>Initial article</p>')
  const [readOnly, setReadOnly] = useState(false)
  const [changes, setChanges] = useState(0)
  return <I18nProvider>
    <label htmlFor="article-body">Article body</label>
    <p id="article-help">Synthetic editor fixture</p>
    <RichEditor id="article-body" aria-label="Article body" aria-describedby="article-help" value={value} readOnly={readOnly}
      onChange={html => { setValue(html); setChanges(count => count + 1) }} />
    <button onClick={() => setValue('<p>Server replacement</p>')}>Replace value</button>
    <button onClick={() => setReadOnly(value => !value)}>Toggle readonly</button>
    <output data-testid="change-count">{changes}</output>
    <output data-testid="stored-html">{value}</output>
  </I18nProvider>
}
createRoot(document.getElementById('root')!).render(<Fixture />)
